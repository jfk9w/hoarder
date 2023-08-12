package tinkoff

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/tinkoff-api"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/jfk9w/hoarder/internal/etl"
)

type parentDesc struct {
	key, value string
}

type update interface {
	entity() string
	parent() []parentDesc
	run(ctx context.Context, log *etl.Logger, client Client, db *gorm.DB) ([]update, error)
}

var updatedAccountTypes = map[string]bool{
	"Current": true,
	"Credit":  true,
	"Saving":  true,
}

type accounts struct {
	overlap   time.Duration
	batchSize int
	phone     string
}

func (u *accounts) entity() string {
	return "accounts"
}

func (u *accounts) parent() []parentDesc {
	return nil
}

func (u *accounts) run(ctx context.Context, log *etl.Logger, client Client, db *gorm.DB) (processes []update, errs error) {
	out, err := client.AccountsLightIb(ctx)
	if log.Error(&errs, err, "failed to get data from api") {
		return
	}

	var (
		entities []Account
		ids      []string
	)

	for _, out := range out {
		id := out.Id
		log := log.WithDesc("accountId", id)

		if accountType := out.AccountType; !updatedAccountTypes[accountType] {
			log.Debug("ignoring type", slog.String("accountType", accountType))
			continue
		}

		entity, err := etl.ToViaJSON[Account](out)
		if log.Error(&errs, err, "entity conversion failed") {
			return
		}

		entity.UserPhone = u.phone

		entities = append(entities, entity)
		ids = append(ids, id)
	}

	if errs = db.Transaction(func(tx *gorm.DB) (errs error) {
		if err := tx.Clauses(etl.Upsert("id")).
			Create(entities).
			Error; log.Error(&errs, err, "failed to update entities in db") {
			return
		}

		if err := tx.Model(new(Account)).
			Where("user_phone = ? and id not in ?", u.phone, ids).
			Update("deleted", true).
			Error; log.Error(&errs, err, "failed to mark deleted entities in db") {
			return
		}

		return
	}); errs != nil {
		return
	}

	log.Info("updated entities in db", slog.Int("count", len(ids)))

	processes = make([]update, len(ids)*2)
	for i, id := range ids {
		processes[2*i] = &statements{
			batchSize: u.batchSize,
			accountId: id,
		}

		processes[2*i+1] = &operations{
			batchSize: u.batchSize,
			overlap:   u.overlap,
			accountId: id,
		}
	}

	processes = append(processes, &receipts{
		batchSize: u.batchSize,
		phone:     u.phone,
	})

	return
}

type statements struct {
	batchSize int
	accountId string
}

func (u *statements) entity() string {
	return "statements"
}

func (u *statements) parent() []parentDesc {
	return []parentDesc{
		{"accountId", u.accountId},
	}
}

func (u *statements) run(ctx context.Context, log *etl.Logger, client Client, db *gorm.DB) (_ []update, errs error) {
	out, err := client.Statements(ctx, &tinkoff.StatementsIn{Account: u.accountId})
	if log.Error(&errs, err, "failed to get data from api") {
		return
	}

	entities, err := etl.ToViaJSON[[]Statement](out)
	if log.Error(&errs, err, "entity conversion failed") {
		return
	}

	for i := range entities {
		entities[i].AccountId = u.accountId
	}

	if err := db.Clauses(etl.Upsert("id")).
		CreateInBatches(entities, u.batchSize).
		Error; log.Error(&errs, err, "failed to update entities in db") {
		return
	}

	log.Info("updated entities in db", slog.Int("count", len(entities)))
	return
}

type operations struct {
	batchSize int
	overlap   time.Duration
	accountId string
}

func (u *operations) entity() string {
	return "operations"
}

func (u *operations) parent() []parentDesc {
	return []parentDesc{
		{"accountId", u.accountId},
	}
}

func (u *operations) run(ctx context.Context, log *etl.Logger, client Client, db *gorm.DB) (_ []update, errs error) {
	var since sql.NullTime
	if err := db.Model(new(Operation)).
		Select("operation_time").
		Where("account_id = ?", u.accountId).
		Order("debiting_time is null desc, operation_time desc").
		Limit(1).
		Scan(&since).
		Error; log.Error(&errs, err, "failed to select latest") {
		return
	}

	start := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	if since.Valid {
		start = since.Time.Add(-u.overlap)
	}

	log = log.With(slog.Time("since", start))

	out, err := client.Operations(ctx, &tinkoff.OperationsIn{Account: u.accountId, Start: start})
	if log.Error(&errs, err, "failed to get data from api") {
		return
	}

	entities, err := etl.ToViaJSON[[]Operation](out)
	if log.Error(&errs, err, "entity conversion failed") {
		return
	}

	if err := db.Clauses(etl.Upsert("id")).
		CreateInBatches(entities, u.batchSize).
		Error; log.Error(&errs, err, "failed to update entities in db") {
		return
	}

	log.Info("updated entities in db", slog.Int("count", len(entities)))
	return
}

type receipts struct {
	batchSize int
	phone     string
}

func (u *receipts) entity() string {
	return "receipts"
}

func (u *receipts) parent() []parentDesc {
	return nil
}

func (u *receipts) run(ctx context.Context, log *etl.Logger, client Client, db *gorm.DB) ([]update, error) {
	it := etl.BatchIterator[int]{
		BatchSize: u.batchSize,
		Log:       log,
		Key:       "offset",
	}

	return nil, it.Run(func(log *etl.Logger, offset int, batchSize int) (nextOffset *int, errs error) {
		var ids []string
		if err := db.Model(new(Operation)).
			Select("operations.id").
			Joins("inner join accounts on operations.account_id = accounts.id").
			Joins("left join receipts on operations.id = receipts.operation_id").
			Where("accounts.user_phone = ? "+
				"and operations.debiting_time is not null "+
				"and operations.has_shopping_receipt "+
				"and receipts.operation_id is null", u.phone).
			Order("operations.debiting_time asc").
			Limit(batchSize).
			Scan(&ids).
			Error; log.Error(&errs, err, "failed to select pending") {
			return
		}

		log.Debug("selected pending", slog.Int("count", len(ids)))

		for _, id := range ids {
			log := log.WithDesc("operationId", id)
			out, err := client.ShoppingReceipt(ctx, &tinkoff.ShoppingReceiptIn{OperationId: id})
			if err != nil {
				if errors.Is(err, tinkoff.ErrNoDataFound) {
					if err := db.Model(new(Operation)).
						Where("id = ?", id).
						Update("has_shopping_receipt", false).
						Error; log.Error(&errs, err, "failed to mark absent entity in db") {
						return
					}

					log.Info("marked absent entity in db")
					continue
				}

				_ = log.Error(&errs, err, "failed to get data from api")
				return
			}

			entity, err := etl.ToViaJSON[Receipt](out.Receipt)
			if log.Error(&errs, err, "entity conversion failed") {
				return
			}

			entity.OperationId = id

			if err := db.Clauses(etl.Upsert("operation_id")).
				Create(&entity).
				Error; log.Error(&errs, err, "failed to update entities in db") {
				return
			}

			log.Info("updated entity in db")
		}

		if len(ids) == batchSize {
			nextOffset = pointer.To(offset + batchSize)
		}

		return
	})
}

type investOperationTypes struct {
	batchSize int
}

func (u *investOperationTypes) entity() string {
	return "investOperationTypes"
}

func (u *investOperationTypes) parent() []parentDesc {
	return nil
}

func (u *investOperationTypes) run(ctx context.Context, log *etl.Logger, client Client, db *gorm.DB) (_ []update, errs error) {
	out, err := client.InvestOperationTypes(ctx)
	if log.Error(&errs, err, "failed to get data from api") {
		return
	}

	var (
		uniqueIds = make(map[string]bool)
		entities  []InvestOperationType
		ids       []string
	)

	for _, out := range out.OperationsTypes {
		id := out.OperationType
		log := log.WithDesc("operationType", id)

		if uniqueIds[id] {
			log.Debug("skipping duplicate")
			continue
		}

		entity, err := etl.ToViaJSON[InvestOperationType](out)
		if log.Error(&errs, err, "entity conversion failed") {
			return
		}

		uniqueIds[id] = true
		entities = append(entities, entity)
		ids = append(ids, id)
	}

	if errs = db.Transaction(func(tx *gorm.DB) (errs error) {
		if err := tx.Clauses(etl.Upsert("operation_type")).
			CreateInBatches(entities, u.batchSize).
			Error; log.Error(&errs, err, "failed to update entities in db") {
			return
		}

		if err := tx.Model(new(InvestOperationType)).
			Where("operation_type not in ?", ids).
			Update("deleted", true).
			Error; log.Error(&errs, err, "failed to mark deleted entities in db") {
			return
		}

		return
	}); errs != nil {
		return
	}

	log.Info("updated entities in db", slog.Int("count", len(entities)))
	return
}

type investAccounts struct {
	clock     based.Clock
	batchSize int
	overlap   time.Duration
	phone     string
}

func (u *investAccounts) entity() string {
	return "investAccounts"
}

func (u *investAccounts) parent() []parentDesc {
	return nil
}

func (u *investAccounts) run(ctx context.Context, log *etl.Logger, client Client, db *gorm.DB) (processes []update, errs error) {
	out, err := client.InvestAccounts(ctx, &tinkoff.InvestAccountsIn{Currency: "RUB"})
	if log.Error(&errs, err, "failed to get data from api") {
		return
	}

	entities, err := etl.ToViaJSON[[]InvestAccount](out.Accounts.List)
	if log.Error(&errs, err, "entity conversion failed") {
		return
	}

	var ids []string
	for i := range entities {
		entity := &entities[i]
		entity.UserPhone = u.phone
		ids = append(ids, entity.Id)
	}

	if errs = db.Transaction(func(tx *gorm.DB) (errs error) {
		if err := tx.Clauses(etl.Upsert("id")).
			Create(entities).
			Error; log.Error(&errs, err, "failed to update entities in db") {
			return
		}

		if err := tx.Model(new(InvestAccount)).
			Where("user_phone = ? and id not in ?", u.phone, ids).
			Update("deleted", true).
			Error; log.Error(&errs, err, "failed to mark deleted entities in db") {
			return
		}

		return
	}); errs != nil {
		return
	}

	log.Info("updated entities in db", slog.Int("count", len(ids)))

	processes = make([]update, len(ids))
	for i, id := range ids {
		processes[i] = &investOperations{
			clock:     u.clock,
			batchSize: u.batchSize,
			overlap:   u.overlap,
			accountId: id,
		}
	}

	return
}

type investOperations struct {
	clock     based.Clock
	batchSize int
	overlap   time.Duration
	accountId string
}

func (u *investOperations) entity() string {
	return "investOperations"
}

func (u *investOperations) parent() []parentDesc {
	return []parentDesc{
		{"investAccountId", u.accountId},
	}
}

func (u *investOperations) run(ctx context.Context, log *etl.Logger, client Client, db *gorm.DB) (_ []update, errs error) {
	var cursor sql.NullString
	if errs = db.Transaction(func(tx *gorm.DB) (errs error) {
		var latestDate sql.NullTime
		if err := db.Model(new(InvestOperation)).
			Select("date").
			Where("invest_account_id = ?", u.accountId).
			Order("date desc").
			Limit(1).
			Scan(&latestDate).
			Error; log.Error(&errs, err, "failed to select latest date") {
			return
		}

		if latestDate.Valid {
			if err := db.Model(new(InvestOperation)).
				Select("cursor").
				Where("invest_account_id = ? and date < ?", u.accountId, latestDate.Time.Add(-u.overlap)).
				Order("date desc").
				Limit(1).
				Scan(&cursor).
				Error; log.Error(&errs, err, "failed to select cursor") {
				return
			}
		}

		return
	}); errs != nil {
		return
	}

	it := etl.BatchIterator[string]{
		BatchSize: u.batchSize,
		Log:       log,
		Key:       "cursor",
		Value:     cursor.String,
	}

	return nil, it.Run(func(log *etl.Logger, cursor string, batchSize int) (nextCursor *string, errs error) {
		in := &tinkoff.InvestOperationsIn{
			From:               time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC),
			To:                 u.clock.Now(),
			BrokerAccountId:    u.accountId,
			OvernightsDisabled: pointer.To(false),
			Limit:              batchSize,
			Cursor:             cursor,
		}

		out, err := client.InvestOperations(ctx, in)
		if log.Error(&errs, err, "failed to get data from api") {
			return
		}

		entities, err := etl.ToViaJSON[[]InvestOperation](out.Items)
		if log.Error(&errs, err, "entity conversion failed") {
			return
		}

		for i := range entities {
			entities[i].InvestAccountId = u.accountId
		}

		if err := db.Clauses(etl.Upsert("internal_id")).
			Create(entities).
			Error; log.Error(&errs, err, "failed to update entities in db") {
			return
		}

		if out.HasNext {
			nextCursor = pointer.To(out.NextCursor)
		}

		log.Info("updated entities in db", slog.Int("count", len(entities)))

		return
	})
}
