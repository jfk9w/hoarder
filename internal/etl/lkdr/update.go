package lkdr

import (
	"context"
	"database/sql"
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/jfk9w-go/lkdr-api"
	"gorm.io/gorm"

	"github.com/jfk9w/hoarder/internal/etl"
	"github.com/jfk9w/hoarder/internal/util"
)

type update interface {
	entity() string
	run(ctx context.Context, log *etl.Logger, client Client, db *gorm.DB) error
}

type receipts struct {
	phone     string
	batchSize int
}

func (u *receipts) entity() string {
	return "receipts"
}

func (u *receipts) run(ctx context.Context, log *etl.Logger, client Client, db *gorm.DB) (errs error) {
	var from sql.NullTime
	if err := db.Model(new(Receipt)).
		Select("receive_date").
		Where("user_phone = ?", u.phone).
		Order("receive_date desc").
		Limit(1).
		Scan(&from).
		Error; log.Error(&errs, err, "failed to select latest receipt date") {
		return
	}

	var dateFrom *lkdr.Date
	if from.Valid {
		dateFrom = pointer.To(lkdr.Date(from.Time))
	}

	it := etl.BatchIterator[int]{
		BatchSize: u.batchSize,
		Log:       log,
		Key:       "offset",
	}

	return it.Run(func(log *etl.Logger, offset int, batchSize int) (nextOffset *int, errs error) {
		in := &lkdr.ReceiptIn{
			DateFrom: dateFrom,
			OrderBy:  "RECEIVE_DATE:ASC",
			Offset:   offset,
			Limit:    batchSize,
		}

		out, err := client.Receipt(ctx, in)
		if log.Error(&errs, err, "failed to get receipts from api") {
			return
		}

		type Entities struct {
			Brands   []Brand   `json:"brands"`
			Receipts []Receipt `json:"receipts"`
		}

		entities, err := etl.ToViaJSON[Entities](out)
		if log.Error(&errs, err, "entity conversion failed") {
			return
		}

		if brands := entities.Brands; len(brands) > 0 {
			if err := db.Clauses(etl.Upsert("id")).
				Create(brands).
				Error; log.Error(&errs, err, "failed to update brands in db") {
				return
			}

			log.Debug("updated brands in db", "count", len(brands))
		}

		if receipts := entities.Receipts; len(receipts) > 0 {
			for i := range receipts {
				receipts[i].UserPhone = u.phone
			}

			if err := db.Clauses(etl.Upsert("key")).
				Create(receipts).
				Error; log.Error(&errs, err, "failed to update receipts in db") {
				return
			}

			log.Debug("updated receipts in db", "count", len(receipts))
		}

		if out.HasMore {
			nextOffset = pointer.To(offset + batchSize)
		}

		log.Debug("batch update finished")

		return
	})
}

type fiscalData struct {
	phone     string
	batchSize int
}

func (u *fiscalData) entity() string {
	return "fiscalData"
}

func (u *fiscalData) run(ctx context.Context, log *etl.Logger, client Client, db *gorm.DB) error {
	it := etl.BatchIterator[int]{
		BatchSize: u.batchSize,
		Log:       log,
		Key:       "offset",
	}

	return it.Run(func(log *etl.Logger, offset int, batchSize int) (nextValue *int, errs error) {
		var pendingReceipts []struct {
			Key           string
			HasFiscalData bool
		}

		if err := db.Model(new(Receipt)).
			Select("receipts.key, fiscal_data.receipt_key is not null as has_fiscal_data").
			Joins("left join fiscal_data on receipts.key = fiscal_data.receipt_key").
			Where("receipts.user_phone = ?", u.phone).
			Order("receive_date asc").
			Offset(offset).
			Limit(batchSize).
			Scan(&pendingReceipts).
			Error; log.Error(&errs, err, "failed to select pending receipts") {
			return
		}

		count := 0
		for _, pendingReceipt := range pendingReceipts {
			if pendingReceipt.HasFiscalData {
				continue
			}

			key := pendingReceipt.Key
			log := log.WithDesc("key", key)
			out, err := client.FiscalData(ctx, &lkdr.FiscalDataIn{Key: key})
			if err != nil {
				msg := "failed to get fiscal data from api"
				if strings.Contains(err.Error(), "Внутреняя ошибка. Попробуйте еще раз.") {
					log.Warn(msg, util.Error(err))
					continue
				}

				_ = log.Error(&errs, err, msg)
				return
			}

			entity, err := etl.ToViaJSON[FiscalData](out)
			if log.Error(&errs, err, "entity conversion failed") {
				return
			}

			entity.Receipt.Key = key

			if err := db.Clauses(etl.Upsert("receipt_key")).
				Create(&entity).
				Error; log.Error(&errs, err, "failed to update fiscal data in db") {
				return
			}

			log.Debug("updated record in db")
			count += 1
		}

		if len(pendingReceipts) == batchSize {
			nextValue = pointer.To(offset + batchSize)
		}

		return
	})
}
