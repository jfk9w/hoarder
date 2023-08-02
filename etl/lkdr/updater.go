package lkdr

import (
	"context"
	"database/sql"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/jfk9w-go/lkdr-api"
	"github.com/jfk9w/hoarder/etl"
	"github.com/jfk9w/hoarder/util"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type updateFunc func(context.Context, *etl.Stats, int) (bool, error)

type updater struct {
	client    Client
	db        *gorm.DB
	phone     string
	batchSize int
	timeout   time.Duration
}

func (u *updater) run(ctx context.Context, stats *etl.Stats, init bool) error {
	var latestReceiptDate sql.NullTime
	if err := u.db.Model(new(Receipt)).
		Select("receive_date").
		Where("user_phone = ?", u.phone).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "receive_date"},
			Desc:   true,
		}).
		Limit(1).
		Scan(&latestReceiptDate).
		Error; err != nil {
		return errors.Wrap(err, "select latest receipt date")
	}

	var receiptDateFrom *lkdr.Date
	if latestReceiptDate.Valid {
		receiptDateFrom = pointer.To(lkdr.Date(latestReceiptDate.Time))
	}

	for _, item := range []struct {
		key string
		fn  updateFunc
	}{
		{
			key: "Receipts",
			fn:  u.updateReceipts(receiptDateFrom),
		},
		{
			key: "Fiscal data",
			fn:  u.updateFiscalData,
		},
	} {
		var (
			hasMore = true
			offset  = 0
			err     error
		)

		for hasMore {
			hasMore, err = item.fn(ctx, stats.Get(item.key, true), offset)
			if err != nil {
				return errors.Wrapf(err, "on %s with offset %d", item.key, offset)
			}

			if init {
				break
			}

			offset += u.batchSize
		}
	}

	return nil
}

func (u *updater) updateFiscalData(ctx context.Context, stats *etl.Stats, offset int) (bool, error) {
	var receiptKeys []string
	if err := u.db.Model(new(Receipt)).
		Select("receipts.key").
		Joins("left join fiscal_data on receipts.key = fiscal_data.receipt_key").
		Where("receipts.user_phone = ? and fiscal_data.receipt_key is null", u.phone).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "receive_date"}}).
		Offset(offset).
		Limit(u.batchSize).
		Scan(&receiptKeys).
		Error; err != nil {
		return false, errors.Wrap(err, "select receipt keys w/o fiscal data")
	}

	for _, key := range receiptKeys {
		var (
			fiscalDataOut *lkdr.FiscalDataOut
			err           error
		)

		util.WithTimeout(ctx, u.timeout, func(ctx context.Context) {
			fiscalDataOut, err = u.client.FiscalData(ctx, &lkdr.FiscalDataIn{Key: key})
		})

		if err != nil {
			stats.Warnf("%s: %s", key, err)
			continue
		}

		fiscalData, err := util.ToViaJSON[FiscalData](fiscalDataOut)
		if err != nil {
			return false, errors.Wrapf(err, "convert fiscal data for receipt %s to entity", key)
		}

		fiscalData.Receipt.Key = key
		for i := range fiscalData.Items {
			fiscalData.Items[i].ReceiptKey = key
			fiscalData.Items[i].Position = i + 1
		}

		if err := u.db.Clauses(util.Upsert("receipt_key")).Create(&fiscalData).Error; err != nil {
			return false, errors.Wrapf(err, "upsert fiscal data %s", key)
		}

		stats.Add(1)
	}

	return len(receiptKeys) == u.batchSize, nil
}

func (u *updater) updateReceipts(receiptDateFrom *lkdr.Date) updateFunc {
	return func(ctx context.Context, stats *etl.Stats, offset int) (bool, error) {
		receiptIn := &lkdr.ReceiptIn{
			DateFrom: receiptDateFrom,
			OrderBy:  "RECEIVE_DATE:ASC",
			Offset:   offset,
			Limit:    u.batchSize,
		}

		var (
			receiptOut *lkdr.ReceiptOut
			err        error
		)

		util.WithTimeout(ctx, u.timeout, func(ctx context.Context) {
			receiptOut, err = u.client.Receipt(ctx, receiptIn)
		})

		if err != nil {
			stats.Error(err)
			return false, nil
		}

		brands, err := util.ToViaJSON[[]Brand](receiptOut.Brands)
		if err != nil {
			return false, errors.Wrap(err, "convert brands to entities")
		}

		if err := u.db.Clauses(util.Upsert("id")).Create(brands).Error; err != nil {
			return false, errors.Wrap(err, "upsert brands")
		}

		receipts, err := util.ToViaJSON[[]Receipt](receiptOut.Receipts)
		if err != nil {
			return false, errors.Wrap(err, "convert receipts to entities")
		}

		for i := range receipts {
			receipt := &receipts[i]
			receipt.UserPhone = u.phone
		}

		if err := u.db.Clauses(util.Upsert("key")).Create(receipts).Error; err != nil {
			return false, errors.Wrap(err, "upsert receipts")
		}

		stats.Add(len(receipts))

		return receiptOut.HasMore, nil
	}
}
