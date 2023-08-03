package tinkoff

import (
	"context"
	"database/sql"
	"time"

	"github.com/AlekSi/pointer"
	"go.uber.org/multierr"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/tinkoff-api"
	"github.com/jfk9w/hoarder/etl"
	"github.com/jfk9w/hoarder/util"
)

type updater struct {
	clock     based.Clock
	client    Client
	db        *gorm.DB
	phone     string
	batchSize int
	overlap   time.Duration
}

func (u *updater) run(ctx context.Context) (errs error) {
	accountIds, err := u.updateAccounts(ctx)
	if err != nil {
		errs = multierr.Append(errs, errors.Wrap(err, "update accounts"))
	} else {
		for _, accountId := range accountIds {
			if err := u.updateOperations(ctx, accountId); err != nil {
				errs = multierr.Append(errs, errors.Wrapf(err, "update operations for account %s", accountId))
			}

			if etl.IsLimited(ctx) {
				break
			}
		}

		if err := u.updateReceipts(ctx); err != nil {
			errs = multierr.Append(errs, errors.Wrap(err, "update receipts"))
		}
	}

	//if err := u.updateInvestOperationTypes(ctx); err != nil {
	//	errs = multierr.Append(errs, errors.Wrap(err, "update invest operation types"))
	//}

	investAccountIds, err := u.updateInvestAccounts(ctx)
	if err != nil {
		errs = multierr.Append(errs, errors.Wrap(err, "update invest accounts"))
	} else {
		for _, investAccountId := range investAccountIds {
			if err := u.updateInvestOperations(ctx, investAccountId); err != nil {
				errs = multierr.Append(errs, errors.Wrapf(err, "update operations for invest account %s", investAccountId))
			}
		}
	}

	return
}

var updatedAccountTypes = map[string]bool{
	"Current": true,
	"Credit":  true,
	"Saving":  true,
}

func (u *updater) updateAccounts(ctx context.Context) ([]string, error) {
	accountsOut, err := u.client.AccountsLightIb(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get")
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
			return nil, errors.Wrapf(err, "convert account %s to entity", accountId)
		}

		account.UserPhone = u.phone

		accounts = append(accounts, account)
		accountIds = append(accountIds, accountId)
	}

	if err := u.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(util.Upsert("id")).Create(accounts).Error; err != nil {
			return errors.Wrap(err, "upsert")
		}

		if err := tx.Model(new(Account)).
			Where("user_phone = ? and id not in ?", u.phone, accountIds).
			Update("deleted", true).
			Error; err != nil {
			return errors.Wrap(err, "mark deleted")
		}

		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "update in db")
	}

	return accountIds, nil
}

func (u *updater) updateOperations(ctx context.Context, accountId string) error {
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
		return errors.Wrap(err, "get")
	}

	operations, err := util.ToViaJSON[[]Operation](operationsOut)
	if err != nil {
		return errors.Wrap(err, "convert to entities")
	}

	if err := u.db.Clauses(util.Upsert("id")).CreateInBatches(operations, u.batchSize).Error; err != nil {
		return errors.Wrap(err, "update in db")
	}

	return nil
}

func (u *updater) updateReceipts(ctx context.Context) error {
	var offset int

	for {
		var operationIds []string
		if err := u.db.Model(new(Operation)).
			Select("operations.id").
			Joins("inner join accounts on operations.account_id = accounts.id").
			Joins("left join receipts on operations.id = receipts.operation_id").
			Where("accounts.user_phone = ? and operations.debiting_time is not null and operations.has_shopping_receipt", u.phone).
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
						return errors.Wrapf(err, "mark operation w/o receipt %s in db", operationId)
					}
				}

				return errors.Wrapf(err, "get for operation %s", operationId)
			}

			receipt, err := util.ToViaJSON[Receipt](shoppingReceiptOut.Receipt)
			if err != nil {
				return errors.Wrapf(err, "convert for operation %s to entity", operationId)
			}

			receipt.OperationId = operationId

			if err := u.db.Clauses(util.Upsert("operation_id")).Create(&receipt).Error; err != nil {
				return errors.Wrapf(err, "update for operation %s in db", operationId)
			}

			if etl.IsLimited(ctx) {
				return nil
			}
		}

		if len(operationIds) < u.batchSize {
			break
		}

		offset += u.batchSize
	}

	return nil
}

func (u *updater) updateInvestOperationTypes(ctx context.Context) error {
	investOperationTypesOut, err := u.client.InvestOperationTypes(ctx)
	if err != nil {
		return errors.Wrap(err, "get")
	}

	investOperationTypes, err := util.ToViaJSON[[]InvestOperationType](investOperationTypesOut.OperationsTypes)
	if err != nil {
		return errors.Wrap(err, "convert to entities")
	}

	var investOperationTypeIds []string
	for _, investOperationType := range investOperationTypes {
		investOperationTypeIds = append(investOperationTypeIds, investOperationType.OperationType)
	}

	if err := u.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(util.Upsert("operation_type")).CreateInBatches(investOperationTypes, u.batchSize).Error; err != nil {
			return errors.Wrap(err, "upsert")
		}

		if err := tx.Model(new(InvestOperationType)).
			Where("operation_type not in ?", investOperationTypeIds).
			Update("deleted", true).
			Error; err != nil {
			return errors.Wrap(err, "mark deleted")
		}

		return nil
	}); err != nil {
		return errors.Wrap(err, "update in db")
	}

	return nil
}

func (u *updater) updateInvestAccounts(ctx context.Context) ([]string, error) {
	investAccountsOut, err := u.client.InvestAccounts(ctx, &tinkoff.InvestAccountsIn{Currency: "RUB"})
	if err != nil {
		return nil, errors.Wrap(err, "get")
	}

	investAccounts, err := util.ToViaJSON[[]InvestAccount](investAccountsOut.Accounts.List)
	if err != nil {
		return nil, errors.Wrap(err, "convert to entities")
	}

	var investAccountIds []string
	for i := range investAccounts {
		investAccount := &investAccounts[i]
		investAccount.UserPhone = u.phone
		investAccountIds = append(investAccountIds, investAccount.Id)
	}

	if err := u.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(util.Upsert("id")).Create(investAccounts).Error; err != nil {
			return errors.Wrap(err, "upsert")
		}

		if err := tx.Model(new(InvestAccount)).
			Where("user_phone = ? and id not in ?", u.phone, investAccountIds).
			Update("deleted", true).
			Error; err != nil {
			return errors.Wrap(err, "mark deleted")
		}

		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "update in db")
	}

	return investAccountIds, nil
}

func (u *updater) updateInvestOperations(ctx context.Context, investAccountId string) error {
	var cursor sql.NullString
	if err := u.db.Model(new(InvestOperation)).
		Select("invest_operations.cursor").
		Joins("inner join invest_accounts on invest_operations.invest_account_id = invest_accounts.id").
		Where("invest_accounts.user_phone = ? and invest_operations.date <= ?", u.phone, u.clock.Now().Add(-u.overlap)).
		Order("invest_operations.date desc").
		Limit(1).
		Scan(&cursor).
		Error; err != nil {
		return errors.Wrap(err, "select cursor")
	}

	for {
		investOperationsIn := &tinkoff.InvestOperationsIn{
			From:               time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC),
			To:                 u.clock.Now(),
			BrokerAccountId:    investAccountId,
			OvernightsDisabled: pointer.To(false),
			Limit:              u.batchSize,
			Cursor:             cursor.String,
		}

		investOperationsOut, err := u.client.InvestOperations(ctx, investOperationsIn)
		if err != nil {
			return errors.Wrapf(err, "get with cursor [%s]", cursor.String)
		}

		investOperations, err := util.ToViaJSON[[]InvestOperation](investOperationsOut.Items)
		if err != nil {
			return errors.Wrap(err, "convert to entities")
		}

		for i := range investOperations {
			investOperations[i].InvestAccountId = investAccountId
		}

		if err := u.db.Clauses(util.Upsert("internal_id")).Create(investOperations).Error; err != nil {
			return errors.Wrap(err, "update in db")
		}

		if !investOperationsOut.HasNext || etl.IsLimited(ctx) {
			break
		}

		cursor.String = investOperationsOut.NextCursor
	}

	return nil
}
