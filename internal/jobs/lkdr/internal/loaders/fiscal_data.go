package loaders

import (
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/jfk9w-go/lkdr-api"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/jobs"
	"github.com/jfk9w/hoarder/internal/jobs/lkdr/internal/entities"
	"github.com/jfk9w/hoarder/internal/logs"
)

type FiscalData struct {
	Phone     string
	BatchSize int
}

func (l FiscalData) TableName() string {
	return new(entities.FiscalData).TableName()
}

func (l FiscalData) Load(ctx jobs.Context, client Client, db database.DB) (_ []Interface, errs error) {
	return nil, jobs.Batch[int]{
		Key:  "offset",
		Size: l.BatchSize,
	}.Run(ctx, fiscalDataBatch{
		phone:  l.Phone,
		client: client,
		db:     db,
	}.load)
}

type fiscalDataBatch struct {
	phone  string
	client Client
	db     database.DB
}

func (l fiscalDataBatch) load(ctx jobs.Context, offset, limit int) (nextOffset *int, errs error) {
	var pendingReceipts []struct {
		Key           string
		HasFiscalData bool
	}

	if err := l.db.WithContext(ctx).
		Model(new(entities.Receipt)).
		Select("receipts.key, fiscal_data.receipt_key is not null as has_fiscal_data").
		Joins("left join fiscal_data on receipts.key = fiscal_data.receipt_key").
		Where("receipts.user_phone = ?", l.phone).
		Order("receive_date asc").
		Offset(offset).
		Limit(limit).
		Scan(&pendingReceipts).
		Error; ctx.Error(&errs, err, "failed to select pending receipts") {
		return
	}

	for _, pendingReceipt := range pendingReceipts {
		if pendingReceipt.HasFiscalData {
			continue
		}

		key := pendingReceipt.Key
		ctx := ctx.With("key", key)

		out, err := l.client.FiscalData(ctx, &lkdr.FiscalDataIn{Key: key})
		if err != nil {
			if lkdr.IsDataNotFound(err) {
				ctx.Warn("fiscal data not found", logs.Error(err))
				continue
			}

			msg := "failed to get data from api"
			if strings.Contains(err.Error(), "Внутреняя ошибка. Попробуйте еще раз.") {
				ctx.Warn(msg, logs.Error(err))
				continue
			}

			_ = ctx.Error(&errs, err, msg)
			return
		}

		entity, err := database.ToViaJSON[entities.FiscalData](out)
		if ctx.Error(&errs, err, "entity conversion failed") {
			return
		}

		entity.ReceiptKey = key

		if err := l.db.WithContext(ctx).
			Upsert(&entity).
			Error; ctx.Error(&errs, err, "failed to update entities in db") {
			return
		}

		ctx.Debug("updated entity in db")
	}

	if len(pendingReceipts) == limit {
		nextOffset = pointer.To(offset + limit)
	}

	return
}
