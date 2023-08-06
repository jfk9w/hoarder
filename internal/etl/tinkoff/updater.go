package tinkoff

import (
	"context"
	"database/sql"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/tinkoff-api"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/jfk9w/hoarder/internal/etl"
)

type updater struct {
	clock           based.Clock
	client          Client
	db              *gorm.DB
	phone           string
	batchSize       int
	overlap         time.Duration
	disableReceipts bool
}

func (u *updater) run(ctx context.Context, log *zap.Logger) (errs error) {
	entityLog := func(name string) *zap.Logger {
		return log.With(zap.String("entity", name))
	}

	accountIds, err := u.accounts(ctx, entityLog("accounts"))
	if !multierr.AppendInto(&errs, errors.Wrap(err, "accounts")) {
		for _, accountId := range accountIds {
			log := entityLog("operations").With(zap.String("accountId", accountId))
			err := u.operations(ctx, log, accountId)
			errs = multierr.Append(errs, errors.Wrapf(err, "account %s: operations", accountId))
		}
	}

	err = u.receipts(ctx, entityLog("receipts"))
	errs = multierr.Append(errs, errors.Wrap(err, "receipts"))

	err = u.investOperationTypes(ctx, entityLog("investOperationTypes"))
	errs = multierr.Append(errs, errors.Wrap(err, "invest operation types"))

	investAccountIds, err := u.investAccounts(ctx, entityLog("invest accounts"))
	if !multierr.AppendInto(&errs, errors.Wrap(err, "invest accounts")) {
		for _, investAccountId := range investAccountIds {
			log := entityLog("investOperations").With(zap.String("investAccountId", investAccountId))
			err := u.investOperations(ctx, log, investAccountId)
			errs = multierr.Append(errs, errors.Wrapf(err, "invest account %s: operations", investAccountId))
		}
	}

	return
}

var updatedAccountTypes = map[string]bool{
	"Current": true,
	"Credit":  true,
	"Saving":  true,
}

func (u *updater) accounts(ctx context.Context, log *zap.Logger) ([]string, error) {
	out, err := u.client.AccountsLightIb(ctx)
	if err != nil {
		log.Error("failed to get", zap.Error(err))
		return nil, errors.New("failed to get")
	}

	var (
		entities []Account
		ids      []string
	)

	for _, out := range out {
		id := out.Id
		log := log.With(zap.String("accountId", id))

		if accountType := out.AccountType; !updatedAccountTypes[accountType] {
			log.Debug("ignoring type", zap.String("accountType", accountType))
			continue
		}

		entity, err := etl.ToViaJSON[Account](out)
		if err != nil {
			log.Error("conversion failed", zap.Error(err))
			return nil, errors.Errorf("%s: conversion failed", id)
		}

		entity.UserPhone = u.phone

		entities = append(entities, entity)
		ids = append(ids, id)
	}

	if err := u.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(etl.Upsert("id")).Create(entities).Error; err != nil {
			log.Error("failed to update in db", zap.Error(err))
			return errors.New("failed to update in db")
		}

		if err := tx.Model(new(Account)).
			Where("user_phone = ? and id not in ?", u.phone, ids).
			Update("deleted", true).
			Error; err != nil {
			log.Error("failed to mark deleted in db", zap.Error(err))
			return errors.New("failed to mark deleted in db")
		}

		return nil
	}); err != nil {
		return nil, err
	}

	log.Info("update completed", zap.Int("count", len(ids)))
	return ids, nil
}

func (u *updater) operations(ctx context.Context, log *zap.Logger, accountId string) error {
	var since sql.NullTime
	if err := u.db.Model(new(Operation)).
		Select("operation_time").
		Where("account_id = ?", accountId).
		Order("debiting_time is null desc, operation_time desc").
		Limit(1).
		Scan(&since).
		Error; err != nil {
		log.Error("failed to select latest", zap.Error(err))
		return errors.New("failed to select latest")
	}

	start := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	if since.Valid {
		start = since.Time.Add(-u.overlap)
	}

	log = log.With(zap.Time("since", start))

	out, err := u.client.Operations(ctx, &tinkoff.OperationsIn{Account: accountId, Start: start})
	if err != nil {
		log.Error("failed to get", zap.Error(err))
		return errors.New("failed to get")
	}

	entities, err := etl.ToViaJSON[[]Operation](out)
	if err != nil {
		log.Error("conversion failed", zap.Error(err))
		return errors.New("conversion failed")
	}

	if err := u.db.Clauses(etl.Upsert("id")).CreateInBatches(entities, u.batchSize).Error; err != nil {
		log.Error("failed to update in db", zap.Error(err))
		return errors.New("failed to update in db")
	}

	log.Info("update completed", zap.Int("count", len(entities)))
	return nil
}

func (u *updater) receipts(ctx context.Context, log *zap.Logger) error {
	var (
		offset int
		count  int
	)

	for {
		log := log.With(zap.Int("offset", offset))

		var ids []string
		if err := u.db.Model(new(Operation)).
			Select("operations.id").
			Joins("inner join accounts on operations.account_id = accounts.id").
			Joins("left join receipts on operations.id = receipts.operation_id").
			Where("accounts.user_phone = ? and operations.debiting_time is not null and operations.has_shopping_receipt", u.phone).
			Order("operations.debiting_time asc").
			Limit(u.batchSize).
			Scan(&ids).
			Error; err != nil {
			log.Error("failed to select pending", zap.Error(err))
			return errors.New("failed to select pending")
		}

		log.Debug("selected pending", zap.Int("count", len(ids)))

		for _, id := range ids {
			log := log.With(zap.String("operationId", id))
			errFn := func(err error, msg string) error {
				if err == nil {
					return nil
				}

				log.Error(msg, zap.Error(err))
				return errors.Errorf("%s: %s", id, msg)
			}

			out, err := u.client.ShoppingReceipt(ctx, &tinkoff.ShoppingReceiptIn{OperationId: id})
			if err != nil {
				if errors.Is(err, tinkoff.ErrNoDataFound) {
					if err := u.db.Model(new(Operation)).
						Where("id = ?", id).
						Update("has_shopping_receipt", false).
						Error; err != nil {
						return errFn(err, "failed to mark absent in db")
					}

					log.Info("marked absent in db")
					continue
				}

				return errFn(err, "get")
			}

			entity, err := etl.ToViaJSON[Receipt](out.Receipt)
			if err != nil {
				return errFn(err, "conversion failed")
			}

			entity.OperationId = id

			if err := u.db.Clauses(etl.Upsert("operation_id")).Create(&entity).Error; err != nil {
				return errFn(err, "failed to update in db")
			}

			log.Debug("partial update completed")
		}

		offset += u.batchSize
		count += len(ids)

		if len(ids) < u.batchSize {
			break
		}
	}

	log.Info("update completed", zap.Int("count", count))
	return nil
}

func (u *updater) investOperationTypes(ctx context.Context, log *zap.Logger) error {
	out, err := u.client.InvestOperationTypes(ctx)
	if err != nil {
		log.Error("failed to get", zap.Error(err))
		return errors.New("failed to get")
	}

	var (
		uniqueIds = make(map[string]bool)
		entities  []InvestOperationType
		ids       []string
	)

	for _, out := range out.OperationsTypes {
		id := out.OperationType
		log := log.With(zap.String("operationType", id))

		if uniqueIds[id] {
			log.Debug("skipping duplicate")
			continue
		}

		entity, err := etl.ToViaJSON[InvestOperationType](out)
		if err != nil {
			log.Error("conversion failed", zap.Error(err))
			return errors.Errorf("%s: conversion failed", id)
		}

		uniqueIds[id] = true
		entities = append(entities, entity)
		ids = append(ids, id)
	}

	if err := u.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(etl.Upsert("operation_type")).
			CreateInBatches(entities, u.batchSize).
			Error; err != nil {
			log.Error("failed to update in db", zap.Error(err))
			return errors.New("failed to update in db")
		}

		if err := tx.Model(new(InvestOperationType)).
			Where("operation_type not in ?", ids).
			Update("deleted", true).
			Error; err != nil {
			log.Error("failed to mark deleted in db")
			return errors.New("failed to mark deleted in db")
		}

		return nil
	}); err != nil {
		return err
	}

	log.Info("update completed", zap.Int("count", len(entities)))
	return nil
}

func (u *updater) investAccounts(ctx context.Context, log *zap.Logger) ([]string, error) {
	out, err := u.client.InvestAccounts(ctx, &tinkoff.InvestAccountsIn{Currency: "RUB"})
	if err != nil {
		log.Error("failed to get", zap.Error(err))
		return nil, errors.New("failed to get")
	}

	entities, err := etl.ToViaJSON[[]InvestAccount](out.Accounts.List)
	if err != nil {
		log.Error("conversion failed", zap.Error(err))
		return nil, errors.New("conversion failed")
	}

	var ids []string
	for i := range entities {
		entity := &entities[i]
		entity.UserPhone = u.phone
		ids = append(ids, entity.Id)
	}

	if err := u.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(etl.Upsert("id")).Create(entities).Error; err != nil {
			log.Error("failed to update in db", zap.Error(err))
			return errors.New("failed to update in db")
		}

		if err := tx.Model(new(InvestAccount)).
			Where("user_phone = ? and id not in ?", u.phone, ids).
			Update("deleted", true).
			Error; err != nil {
			log.Error("failed to mark deleted in db", zap.Error(err))
			return errors.Wrap(err, "failed to mark deleted in db")
		}

		return nil
	}); err != nil {
		return nil, err
	}

	log.Info("update completed", zap.Int("count", len(ids)))
	return ids, nil
}

func (u *updater) investOperations(ctx context.Context, log *zap.Logger, investAccountId string) error {
	var (
		cursor sql.NullString
		count  int
	)

	if err := u.db.Model(new(InvestOperation)).
		Select("invest_operations.cursor").
		Joins("inner join invest_accounts on invest_operations.invest_account_id = invest_accounts.id").
		Where("invest_accounts.user_phone = ? and invest_operations.date <= ?", u.phone, u.clock.Now().Add(-u.overlap)).
		Order("invest_operations.date desc").
		Limit(1).
		Scan(&cursor).
		Error; err != nil {
		log.Error("failed to select cursor", zap.Error(err))
		return errors.New("failed to select cursor")
	}

	for {
		log := log.With(zap.String("cursor", cursor.String))
		errFn := func(err error, msg string) error {
			if err == nil {
				return nil
			}

			log.Error(msg, zap.Error(err))
			if cursor.Valid {
				return errors.Errorf("%s: %s", cursor.String, msg)
			} else {
				return errors.New(msg)
			}
		}

		in := &tinkoff.InvestOperationsIn{
			From:               time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC),
			To:                 u.clock.Now(),
			BrokerAccountId:    investAccountId,
			OvernightsDisabled: pointer.To(false),
			Limit:              u.batchSize,
			Cursor:             cursor.String,
		}

		out, err := u.client.InvestOperations(ctx, in)
		if err != nil {
			return errFn(err, "failed to get")
		}

		entities, err := etl.ToViaJSON[[]InvestOperation](out.Items)
		if err != nil {
			return errFn(err, "conversion failed")
		}

		for i := range entities {
			entities[i].InvestAccountId = investAccountId
		}

		if err := u.db.Clauses(etl.Upsert("internal_id")).Create(entities).Error; err != nil {
			return errFn(err, "failed to update in db")
		}

		cursor.String = out.NextCursor
		count += len(entities)
		log.Debug("partial update completed")

		if !out.HasNext {
			break
		}
	}

	log.Info("update completed", zap.Int("count", count))
	return nil
}
