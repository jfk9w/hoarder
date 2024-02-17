package loaders

import (
	"errors"

	"github.com/AlekSi/pointer"
	"github.com/jfk9w-go/tinkoff-api"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/jobs"
	. "github.com/jfk9w/hoarder/internal/jobs/tinkoff/internal/entities"
)

type receipts struct {
	phone     string
	batchSize int
}

func (l receipts) TableName() string {
	return new(Receipt).TableName()
}

func (l receipts) Load(ctx jobs.Context, client Client, db database.DB) ([]Interface, error) {
	return nil, jobs.Batch[int]{
		Size: l.batchSize,
		Key:  "offset",
	}.Run(ctx, receiptsBatch{
		phone:  l.phone,
		client: client,
		db:     db,
	}.load)
}

type receiptsBatch struct {
	phone  string
	client Client
	db     database.DB
}

func (l receiptsBatch) load(ctx jobs.Context, offset int, limit int) (nextOffset *int, errs error) {
	var ids []string
	if err := l.db.WithContext(ctx).Model(new(Operation)).
		Select("operations.id").
		Joins("inner join accounts on operations.account_id = accounts.id").
		Joins("left join receipts on operations.id = receipts.operation_id").
		Where("accounts.user_phone = ? "+
			"and operations.debiting_time is not null "+
			"and operations.has_shopping_receipt "+
			"and receipts.operation_id is null", l.phone).
		Order("operations.debiting_time asc").
		Limit(limit).
		Scan(&ids).
		Error; ctx.Error(&errs, err, "failed to select pending") {
		return
	}

	ctx.Debug("selected pending", "count", len(ids))

	for _, id := range ids {
		ctx := ctx.With("operation_id", id)
		out, err := l.client.ShoppingReceipt(ctx, &tinkoff.ShoppingReceiptIn{OperationId: id})
		if err != nil {
			if errors.Is(err, tinkoff.ErrNoDataFound) {
				if err := l.db.WithContext(ctx).Model(new(Operation)).
					Where("id = ?", id).
					Update("has_shopping_receipt", false).
					Error; ctx.Error(&errs, err, "failed to mark absent entity in db") {
					return
				}

				ctx.Info("marked absent entity in db")
				continue
			}

			_ = ctx.Error(&errs, err, "failed to get data from api")
			return
		}

		entity, err := database.ToViaJSON[Receipt](out.Receipt)
		if ctx.Error(&errs, err, "entity conversion failed") {
			return
		}

		entity.OperationId = id

		if err := l.db.WithContext(ctx).
			Upsert(&entity).
			Error; ctx.Error(&errs, err, "failed to update entities in db") {
			return
		}

		ctx.Info("updated entity in db")
	}

	if len(ids) == limit {
		nextOffset = pointer.To(offset + limit)
	}

	return
}
