package tinkoff

import (
	"context"
	"database/sql"
	"time"

	"github.com/jfk9w-go/tinkoff-api"
	"github.com/jfk9w/hoarder/etl"
	"github.com/jfk9w/hoarder/util"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type updater struct {
	client    Client
	db        *gorm.DB
	phone     string
	batchSize int
	overlap   time.Duration
}

func (u *updater) run(ctx context.Context, stats *etl.Stats) error {
	if err := u.updateAccounts(ctx, stats.Get("Accounts", true)); err != nil {
		return errors.Wrap(err, "update accounts")
	}

	var accountIds []string
	if err := u.db.Model(new(Account)).
		Select("id").
		Where("user_phone = ? and not deleted", u.phone).
		Order("id").
		Scan(&accountIds).
		Error; err != nil {
		return errors.Wrap(err, "select accounts")
	}

	for _, accountId := range accountIds {
		if err := u.updateOperations(ctx, stats.Get("Operations", true), accountId); err != nil {
			return errors.Wrapf(err, "update operations for account %s", accountId)
		}

		if err := u.updateReceipts(ctx, stats.Get("Receipts", true), accountId); err != nil {
			return errors.Wrapf(err, "update receipts for account %s", accountId)
		}
	}

	return nil
}

var updatedAccountTypes = map[string]bool{
	"Current": true,
	"Credit":  true,
	"Saving":  true,
}

func (u *updater) updateAccounts(ctx context.Context, stats *etl.Stats) error {
	accountsOut, err := u.client.AccountsLightIb(ctx)
	if err != nil {
		stats.Error(err)
		return nil
	}

	var (
		accounts   []Account
		accountIds []string
	)

	for _, account := range accountsOut {
		if !updatedAccountTypes[account.AccountType] {
			continue
		}

		accountId := account.Id

		account, err := util.ToViaJSON[Account](account)
		if err != nil {
			return errors.Wrapf(err, "convert account %s to entity", accountId)
		}

		account.UserPhone = u.phone

		accounts = append(accounts, account)
		accountIds = append(accountIds, accountId)
	}

	if err := u.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(util.Upsert("id")).Create(accounts).Error; err != nil {
			return errors.Wrap(err, "upsert accounts")
		}

		if err := tx.Model(new(Account)).
			Where("user_phone = ? and id not in ?", u.phone, accountIds).
			Update("deleted", true).
			Error; err != nil {
			return errors.Wrap(err, "mark deleted accounts")
		}

		return nil
	}); err != nil {
		return errors.Wrap(err, "update accounts in db")
	}

	stats.Add(len(accounts))

	return nil
}

func (u *updater) updateOperations(ctx context.Context, stats *etl.Stats, accountId string) error {
	var latestOperationTime sql.NullTime
	if err := u.db.Model(new(Operation)).
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
		start = latestOperationTime.Time.Add(-u.overlap)
	}

	operationsIn := &tinkoff.OperationsIn{
		Account: accountId,
		Start:   start,
	}

	operationsOut, err := u.client.Operations(ctx, operationsIn)
	if err != nil {
		stats.Error(err)
		return nil
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

	if err := u.db.Clauses(util.Upsert("id")).CreateInBatches(operations, u.batchSize).Error; err != nil {
		return errors.Wrap(err, "upsert operations")
	}

	stats.Add(len(operations))

	return nil
}

func (u *updater) updateReceipts(ctx context.Context, stats *etl.Stats, accountId string) error {
	var offset int

	for !stats.IsError() {
		var operationIds []string
		if err := u.db.Model(new(Operation)).
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

			shoppingReceiptOut, err := u.client.ShoppingReceipt(ctx, shoppingReceiptIn)
			if err != nil {
				if errors.Is(err, tinkoff.ErrNoDataFound) {
					if err := u.db.Model(new(Operation)).
						Where("id = ?", operationId).
						Update("has_shopping_receipt", false).
						Error; err != nil {
						return errors.Wrapf(err, "update operation %s in db", operationId)
					}
				}

				stats.Error(err)
				return nil
			}

			receipt, err := util.ToViaJSON[Receipt](shoppingReceiptOut.Receipt)
			if err != nil {
				return errors.Wrapf(err, "convert receipt for operation %s to entity", operationId)
			}

			receipt.OperationId = operationId

			if err := u.db.Clauses(util.Upsert("operation_id")).Create(&receipt).Error; err != nil {
				return errors.Wrapf(err, "create receipt for operation %s in db", operationId)
			}

			stats.Add(1)
		}

		if len(operationIds) < u.batchSize {
			break
		}

		offset += u.batchSize
	}

	return nil
}
