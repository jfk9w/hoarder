package loaders

import (
	"time"

	"github.com/jfk9w-go/tinkoff-api/v2"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/jobs"
	. "github.com/jfk9w/hoarder/internal/jobs/tinkoff/internal/entities"
)

type operations struct {
	accountId string
	batchSize int
	overlap   time.Duration
}

func (l operations) TableName() string {
	return new(Operation).TableName()
}

func (l operations) Load(ctx jobs.Context, client Client, db database.DB) (ls []Interface, errs error) {
	ctx = ctx.With("account_id", l.accountId)

	var since []struct {
		Debited bool
		MinTime time.Time
		MaxTime time.Time
	}

	if err := db.WithContext(ctx).
		Model(new(Operation)).
		Select("debiting_time is not null as debited, min(operation_time) as min_time, max(operation_time) as max_time").
		Where("account_id = ? and status = ?", l.accountId, "OK").
		Group("debited").
		Order("debited").
		Scan(&since).
		Error; ctx.Error(&errs, err, "failed to select latest") {
		return
	}

	start := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, row := range since {
		if row.Debited {
			start = row.MaxTime
		} else {
			start = row.MinTime
		}

		start = start.Add(-l.overlap)
		break //nolint:all
	}

	ctx = ctx.With("since", start)

	out, err := client.Operations(ctx, &tinkoff.OperationsIn{Account: l.accountId, Start: start})
	if ctx.Error(&errs, err, "failed to get data from api") {
		return
	}

	if len(out) == 0 {
		return
	}

	entities, err := database.ToViaJSON[[]Operation](out)
	if ctx.Error(&errs, err, "entity conversion failed") {
		return
	}

	if errs = db.WithContext(ctx).Transaction(func(tx database.DB) (errs error) {
		if err := tx.
			Where("account_id = ? and status = ? and debiting_time is null and operation_time >= ?", l.accountId, "OK", start).
			Delete(new(Operation)).
			Error; ctx.Error(&errs, err, "failed to delete non-debited operations") {
			return
		}

		if err := tx.UpsertInBatches(entities, l.batchSize).Error; ctx.Error(&errs, err, "failed to update entities in db") {
			return
		}

		return
	}); errs != nil {
		return
	}

	ctx.Info("updated entities in db", "count", len(entities))
	return
}
