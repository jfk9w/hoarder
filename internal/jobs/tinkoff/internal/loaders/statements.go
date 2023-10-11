package loaders

import (
	"github.com/jfk9w-go/tinkoff-api"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/jobs"
	. "github.com/jfk9w/hoarder/internal/jobs/tinkoff/internal/entities"
)

type statements struct {
	accountId string
	batchSize int
}

func (l statements) TableName() string {
	return new(Statement).TableName()
}

func (l statements) Load(ctx jobs.Context, client Client, db database.DB) (ls []Interface, errs error) {
	ctx = ctx.With("account_id", l.accountId)

	out, err := client.Statements(ctx, &tinkoff.StatementsIn{Account: l.accountId})
	if ctx.Error(&errs, err, "failed to get data from api") {
		return
	}

	entities, err := database.ToViaJSON[[]Statement](out)
	if ctx.Error(&errs, err, "entity conversion failed") {
		return
	}

	for i := range entities {
		entities[i].AccountId = l.accountId
	}

	if err := db.WithContext(ctx).
		UpsertInBatches(entities, l.batchSize).
		Error; ctx.Error(&errs, err, "failed to update entities in db") {
		return
	}

	ctx.Info("updated entities in db", "count", len(entities))
	return
}
