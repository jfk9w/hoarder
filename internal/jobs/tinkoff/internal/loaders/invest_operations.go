package loaders

import (
	"database/sql"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/tinkoff-api"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/jobs"
	. "github.com/jfk9w/hoarder/internal/jobs/tinkoff/internal/entities"
)

type investOperations struct {
	accountId string
	batchSize int
	overlap   time.Duration
	clock     based.Clock
}

func (l investOperations) TableName() string {
	return new(InvestOperation).TableName()
}

func (l investOperations) Load(ctx jobs.Context, client Client, db database.DB) (ls []Interface, errs error) {
	ctx = ctx.With("account_id", l.accountId)

	var latestDate sql.NullTime
	if err := db.WithContext(ctx).Model(new(InvestOperation)).
		Select("date").
		Where("invest_account_id = ?", l.accountId).
		Order("date desc").
		Limit(1).
		Scan(&latestDate).
		Error; ctx.Error(&errs, err, "failed to select latest date") {
		return
	}

	from := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)
	if latestDate.Valid {
		from = latestDate.Time.Add(-l.overlap)
	}

	return nil, jobs.Batch[string]{
		Key:  "cursor",
		Size: l.batchSize,
	}.Run(ctx, investOperationsBatch{
		accountId: l.accountId,
		client:    client,
		db:        db,
		from:      from,
		to:        l.clock.Now(),
	}.load)
}

type investOperationsBatch struct {
	accountId string
	client    Client
	db        database.DB
	from      time.Time
	to        time.Time
}

func (l investOperationsBatch) load(ctx jobs.Context, cursor string, limit int) (nextCursor *string, errs error) {
	in := &tinkoff.InvestOperationsIn{
		From:               l.from,
		To:                 l.to,
		BrokerAccountId:    l.accountId,
		OvernightsDisabled: pointer.To(false),
		Limit:              limit,
		Cursor:             cursor,
	}

	out, err := l.client.InvestOperations(ctx, in)
	if ctx.Error(&errs, err, "failed to get data from api") {
		return
	}

	entities, err := database.ToViaJSON[[]InvestOperation](out.Items)
	if ctx.Error(&errs, err, "entity conversion failed") {
		return
	}

	for i := range entities {
		entities[i].InvestAccountId = l.accountId
	}

	if err := l.db.WithContext(ctx).
		Upsert(entities).
		Error; ctx.Error(&errs, err, "failed to update entities in db") {
		return
	}

	if out.HasNext {
		nextCursor = pointer.To(out.NextCursor)
	}

	ctx.Info("updated entities in db", "count", len(entities))

	return
}
