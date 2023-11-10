package loaders

import (
	"time"

	"github.com/jfk9w-go/tinkoff-api"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/jobs"
	. "github.com/jfk9w/hoarder/internal/jobs/tinkoff/internal/entities"
)

type InvestAccounts struct {
	Phone     string
	BatchSize int
	Overlap   time.Duration
	Now       time.Time
}

func (l InvestAccounts) TableName() string {
	return new(InvestAccount).TableName()
}

func (l InvestAccounts) Load(ctx jobs.Context, client Client, db database.DB) (ls []Interface, errs error) {
	out, err := client.InvestAccounts(ctx, &tinkoff.InvestAccountsIn{Currency: "RUB"})
	if ctx.Error(&errs, err, "failed to get data from api") {
		return
	}

	entities, err := database.ToViaJSON[[]InvestAccount](out.Accounts.List)
	if ctx.Error(&errs, err, "entity conversion failed") {
		return
	}

	var ids []string
	for i := range entities {
		entity := &entities[i]
		entity.UserPhone = l.Phone
		ids = append(ids, entity.Id)
	}

	if errs = db.WithContext(ctx).Transaction(func(tx database.DB) (errs error) {
		if err := tx.Upsert(entities).Error; ctx.Error(&errs, err, "failed to update entities in db") {
			return
		}

		if err := tx.Model(new(InvestAccount)).
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
		ls = append(ls, investOperations{
			accountId: id,
			batchSize: l.BatchSize,
			overlap:   l.Overlap,
			now:       l.Now,
		})
	}

	return
}
