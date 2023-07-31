package tinkoff

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/tinkoff-api"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/jfk9w/hoarder/database"
	"github.com/jfk9w/hoarder/util"
)

type Processor struct {
	clients map[string]map[string]*based.Lazy[Client]
	db      *based.Lazy[*gorm.DB]
}

func NewProcessor(cfg Config, clock based.Clock) *Processor {
	db := &based.Lazy[*gorm.DB]{
		Fn: func(ctx context.Context) (*gorm.DB, error) {
			db, err := database.Open(cfg.DB)
			if err != nil {
				return nil, errors.Wrap(err, "open db connection")
			}

			if err := db.WithContext(ctx).AutoMigrate(
				new(User),
				new(Session),
				new(Currency),
				new(Card),
				new(Account),
				new(Category),
				new(Location),
				new(LoyaltyBonus),
				new(SpendingCategory),
				new(Brand),
				new(AdditionalInfo),
				new(LoyaltyPayment),
				new(Payment),
				new(Subgroup),
				new(Operation),
				new(ReceiptItem),
				new(Receipt),
			); err != nil {
				return nil, errors.Wrap(err, "migrate db tables")
			}

			return db, nil
		},
	}

	//sessionStorage := &sessionStorage{db: db}
	clients := make(map[string]map[string]*based.Lazy[Client])
	for username, credentials := range cfg.Users {
		clients[username] = make(map[string]*based.Lazy[Client])
		clients := clients[username]
		for _, credential := range credentials {
			credential := credential
			clients[credential.Phone] = &based.Lazy[Client]{
				Fn: func(ctx context.Context) (Client, error) {
					//return tinkoff.ClientBuilder{
					//	Clock: clock,
					//	Credential: tinkoff.Credential{
					//		Phone:    credential.Phone,
					//		Password: credential.Password,
					//	},
					//	SessionStorage: sessionStorage,
					//}.Build(ctx)
					return new(mockClient), nil
				},
			}
		}
	}

	return &Processor{
		clients: clients,
		db:      db,
	}
}

func (p *Processor) Process(ctx context.Context, username string) error {
	clients, ok := p.clients[username]
	if !ok {
		return nil
	}

	db, err := p.db.Get(ctx)
	if err != nil {
		return errors.Wrap(err, "get db handle")
	}

	db = db.WithContext(ctx)

	for phone, client := range clients {
		client, err := client.Get(ctx)
		if err != nil {
			return errors.Wrapf(err, "get client for %s", phone)
		}

		user := User{
			Name:  username,
			Phone: phone,
		}

		if err := db.Clauses(util.Upsert("phone")).Create(user).Error; err != nil {
			return errors.Wrapf(err, "create user %s:%s in db", username, phone)
		}

		if err := updateData(ctx, client, db, phone); err != nil {
			return errors.Wrapf(err, "update data for %s", phone)
		}
	}

	return nil
}

var allowedAccountTypes = map[string]bool{
	"Current": true,
	"Credit":  true,
	"Saving":  true,
}

func updateData(ctx context.Context, client Client, db *gorm.DB, phone string) error {
	accountsOut, err := client.AccountsLightIb(ctx)
	if err != nil {
		return errors.Wrap(err, "get accounts")
	}

	var accounts []Account
	for _, account := range accountsOut {
		if !allowedAccountTypes[account.AccountType] {
			continue
		}

		accountId := account.Id

		account, err := util.ToViaJSON[Account](account)
		if err != nil {
			return errors.Wrapf(err, "convert account %s to entity", accountId)
		}

		account.UserPhone = phone

		accounts = append(accounts, account)
	}

	if err := db.Clauses(util.Upsert("id")).Create(accounts).Error; err != nil {
		return errors.Wrap(err, "upsert accounts")
	}

	for _, account := range accounts {
		if err := updateOperations(ctx, client, db, account.Id); err != nil {
			return errors.Wrapf(err, "update operations for account %s", account.Id)
		}

		if err := updateReceipts(ctx, client, db, account.Id); err != nil {
			return errors.Wrapf(err, "update receipts for account %s", account.Id)
		}
	}

	return nil
}

func updateOperations(ctx context.Context, client Client, db *gorm.DB, accountId string) error {
	var latestOperationTime sql.NullTime
	if err := db.Model(new(Operation)).
		Select("operation_time").
		Where("account_id = ? and debiting_time is null", accountId).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "operation_time"}}).
		Limit(1).
		Scan(&latestOperationTime).
		Error; err != nil {
		return errors.Wrap(err, "select latest operation time")
	}

	start := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	if latestOperationTime.Valid {
		start = latestOperationTime.Time
	}

	operationsIn := &tinkoff.OperationsIn{
		Account: accountId,
		Start:   start,
	}

	operationsOut, err := client.Operations(ctx, operationsIn)
	if err != nil {
		return errors.Wrap(err, "get operations")
	}

	operations, err := util.ToViaJSON[[]Operation](operationsOut)
	if err != nil {
		return errors.Wrap(err, "convert operations to entities")
	}

	for _, operation := range operations {
		for i := range operation.Locations {
			operation.Locations[i].Position = i + 1
		}

		for i := range operation.AdditionalInfo {
			operation.AdditionalInfo[i].Position = i + 1
		}

		for i := range operation.LoyaltyPayment {
			operation.LoyaltyPayment[i].Position = i + 1
		}

		for i := range operation.LoyaltyBonus {
			operation.LoyaltyBonus[i].Position = i + 1
		}
	}

	if err := db.Clauses(util.Upsert("id")).CreateInBatches(operations, 100).Error; err != nil {
		return errors.Wrap(err, "upsert operations")
	}

	return nil
}

func updateReceipts(ctx context.Context, client Client, db *gorm.DB, accountId string) error {
	var offset int

	for {
		var operationIds []string
		if err := db.Model(new(Operation)).
			Select("operations.id").
			Joins("left join receipts on operations.id = receipts.operation_id").
			Where("operations.account_id = ? and operations.debiting_time is not null and operations.has_shopping_receipt", accountId).
			Order("operations.debiting_time asc").
			Offset(offset).
			Limit(1000).
			Scan(&operationIds).
			Error; err != nil {
			return errors.Wrap(err, "select operations with receipts")
		}

		for _, operationId := range operationIds {
			shoppingReceiptIn := &tinkoff.ShoppingReceiptIn{
				OperationId: operationId,
			}

			shoppingReceiptOut, err := client.ShoppingReceipt(ctx, shoppingReceiptIn)
			if err != nil {
				if errors.Is(err, tinkoff.ErrNoDataFound) {
					if err := db.Model(new(Operation)).
						Where("id = ?", operationId).
						Update("has_shopping_receipt", false).
						Error; err != nil {
						return errors.Wrapf(err, "update operation %s in db", operationId)
					}
				}

				err = errors.Wrapf(err, "get shopping receipt for %s", operationId)
				fmt.Println(err)

				continue
			}

			receipt, err := util.ToViaJSON[Receipt](shoppingReceiptOut.Receipt)
			if err != nil {
				return errors.Wrapf(err, "convert receipt for operation %s to entity", operationId)
			}

			receipt.OperationId = operationId

			if err := db.Clauses(util.Upsert("operation_id")).Create(&receipt).Error; err != nil {
				return errors.Wrapf(err, "create receipt for operation %s in db", operationId)
			}
		}

		if len(operationIds) < 1000 {
			break
		}

		offset += 1000
	}

	return nil
}
