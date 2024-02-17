package loaders

import (
	"time"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/jobs"
	. "github.com/jfk9w/hoarder/internal/jobs/tinkoff/internal/entities"
)

var supportedAccountTypes = map[string]bool{
	"Current": true,
	"Credit":  true,
	"Saving":  true,
}

type Accounts struct {
	Phone        string
	BatchSize    int
	Overlap      time.Duration
	WithReceipts bool
}

func (l Accounts) TableName() string {
	return new(Account).TableName()
}

func (l Accounts) Load(ctx jobs.Context, client Client, db database.DB) (ls []Interface, errs error) {
	out, err := client.AccountsLightIb(ctx)
	if ctx.Error(&errs, err, "failed to get data from api") {
		return
	}

	var (
		entities []Account
		ids      []string
	)

	for _, out := range out {
		ctx := ctx.With("account_id", out.Id)
		if !supportedAccountTypes[out.AccountType] {
			ctx.Debug("account type is not supported", "account_type", out.AccountType)
			continue
		}

		entity, err := database.ToViaJSON[Account](out)
		if ctx.Error(&errs, err, "entity conversion failed") {
			return
		}

		entity.UserPhone = l.Phone
		entities = append(entities, entity)
		ids = append(ids, out.Id)
	}

	if errs = db.WithContext(ctx).Transaction(func(tx database.DB) (errs error) {
		if err := tx.Upsert(entities).Error; ctx.Error(&errs, err, "failed to update entities in db") {
			return
		}

		if err := tx.Model(new(Account)).
			Where("user_phone = ? and id not in ?", l.Phone, ids).
			Update("deleted", true).
			Error; ctx.Error(&errs, err, "failed to mark deleted entities in db") {
			return
		}

		return
	}); errs != nil {
		return
	}

	ctx.Info("updated entities in db", "count", len(ids))

	for _, id := range ids {
		ls = append(ls,
			accountRequisites{accountId: id},
			statements{accountId: id, batchSize: l.BatchSize},
			operations{accountId: id, batchSize: l.BatchSize, overlap: l.Overlap})
	}

	if l.WithReceipts {
		ls = append(ls, receipts{phone: l.Phone, batchSize: l.BatchSize})
	}

	return
}
