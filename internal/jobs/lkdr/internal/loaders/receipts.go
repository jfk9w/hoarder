package loaders

import (
	"database/sql"
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/jfk9w-go/lkdr-api"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/jobs"
	"github.com/jfk9w/hoarder/internal/jobs/lkdr/internal/entities"
	"github.com/jfk9w/hoarder/internal/logs"
)

type Receipts struct {
	Phone     string
	BatchSize int
}

func (l Receipts) TableName() string {
	return new(entities.Receipt).TableName()
}

func (l Receipts) Load(ctx jobs.Context, client Client, db database.DB) (_ []Interface, errs error) {
	var from sql.NullTime
	if err := db.WithContext(ctx).
		Model(new(entities.Receipt)).
		Select("receive_date").
		Where("user_phone = ?", l.Phone).
		Order("receive_date desc").
		Limit(1).
		Scan(&from).
		Error; ctx.Error(&errs, err, "failed to select latest date") {
		return
	}

	var dateFrom *lkdr.Date
	if from.Valid {
		dateFrom = pointer.To(lkdr.Date(from.Time))
	}

	return nil, jobs.Batch[int]{
		Key:  "offset",
		Size: l.BatchSize,
	}.Run(ctx, receiptsBatch{
		phone:    l.Phone,
		client:   client,
		db:       db,
		dateFrom: dateFrom,
	}.load)
}

type receiptsBatch struct {
	phone    string
	client   Client
	db       database.DB
	dateFrom *lkdr.Date
}

func (l receiptsBatch) load(ctx jobs.Context, offset int, limit int) (nextOffset *int, errs error) {
	in := &lkdr.ReceiptIn{
		DateFrom: l.dateFrom,
		OrderBy:  "RECEIVE_DATE:ASC",
		Offset:   offset,
		Limit:    limit,
	}

	out, err := l.client.Receipt(ctx, in)
	if err != nil {
		msg := "failed to get data from api"
		if strings.Contains(err.Error(), "Внутреняя ошибка. Попробуйте еще раз") {
			ctx.Warn(msg, logs.Error(err))
			return
		}

		_ = ctx.Error(&errs, err, msg)
		return
	}

	type Entities struct {
		Brands   []entities.Brand   `json:"brands"`
		Receipts []entities.Receipt `json:"receipts"`
	}

	entities, err := database.ToViaJSON[Entities](out)
	if ctx.Error(&errs, err, "entity conversion failed") {
		return
	}

	if brands := entities.Brands; len(brands) > 0 {
		if err := l.db.WithContext(ctx).
			Upsert(brands).
			Error; ctx.Error(&errs, err, "failed to update brands in db") {
			return
		}

		ctx.Debug("updated brands in db", "count", len(brands))
	}

	if receipts := entities.Receipts; len(receipts) > 0 {
		for i := range receipts {
			receipts[i].UserPhone = l.phone
		}

		if err := l.db.WithContext(ctx).
			Upsert(receipts).
			Error; ctx.Error(&errs, err, "failed to update receipts in db") {
			return
		}

		ctx.Debug("updated receipts in db", "count", len(receipts))
	}

	if out.HasMore {
		nextOffset = pointer.To(offset + limit)
	}

	return
}
