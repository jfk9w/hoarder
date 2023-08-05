package lkdr

import (
	"context"
	"database/sql"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/jfk9w-go/lkdr-api"

	"github.com/jfk9w/hoarder/internal/util"
)

type updateFunc func(ctx context.Context, log *zap.Logger, offset int) (bool, error)

type updater struct {
	client    Client
	db        *gorm.DB
	phone     string
	batchSize int
	timeout   time.Duration
}

func (u *updater) run(ctx context.Context, log *zap.Logger) (errs error) {
	var latestReceiptDate sql.NullTime
	if err := u.db.Model(new(Receipt)).
		Select("receive_date").
		Where("user_phone = ?", u.phone).
		Order("receive_date desc").
		Limit(1).
		Scan(&latestReceiptDate).
		Error; err != nil {
		log.Error("select latest receipt", zap.Error(err))
		return errors.New("select latest receipt")
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
			key: "receipts",
			fn:  u.receipts(receiptDateFrom),
		},
		{
			key: "fiscalData",
			fn:  u.fiscalData,
		},
	} {
		var (
			log     = log.With(zap.String("entity", item.key))
			hasMore = true
			offset  = 0
			err     error
		)

		for hasMore {
			log := log.With(zap.Int("offset", offset))
			hasMore, err = item.fn(ctx, log, offset)
			if err != nil {
				errs = multierr.Append(errs, err)
			}

			if !hasMore {
				break
			}

			offset += u.batchSize
		}

		log.Debug("update completed")
	}

	return
}

func (u *updater) fiscalData(ctx context.Context, log *zap.Logger, offset int) (hasMore bool, errs error) {
	var keys []string
	if err := u.db.Model(new(Receipt)).
		Select("receipts.key").
		Joins("left join fiscal_data on receipts.key = fiscal_data.receipt_key").
		Where("receipts.user_phone = ? and fiscal_data.receipt_key is null", u.phone).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "receive_date"}}).
		Offset(offset).
		Limit(u.batchSize).
		Scan(&keys).
		Error; err != nil {
		log.Error("select pending keys", zap.Error(err))
		return false, errors.New("select pending keys")
	}

	for _, key := range keys {
		var (
			log   = log.With(zap.String("key", key))
			errFn = func(err error, level zapcore.Level, msg string) error {
				if err == nil {
					return nil
				}

				log.Log(level, msg, zap.Error(err))
				return errors.Errorf("%s: %s", key, msg)
			}

			out *lkdr.FiscalDataOut
			err error
		)

		util.WithTimeout(ctx, u.timeout, func(ctx context.Context) {
			out, err = u.client.FiscalData(ctx, &lkdr.FiscalDataIn{Key: key})
		})

		if err := errFn(err, zapcore.WarnLevel, "failed to get"); multierr.AppendInto(&errs, err) {
			continue
		}

		entity, err := util.ToViaJSON[FiscalData](out)
		if err := errFn(err, zapcore.ErrorLevel, "conversion failed"); multierr.AppendInto(&errs, err) {
			return
		}

		entity.Receipt.Key = key

		err = u.db.Clauses(util.Upsert("receipt_key")).Create(&entity).Error
		if err := errFn(err, zapcore.ErrorLevel, "failed to update in db"); multierr.AppendInto(&errs, err) {
			return
		}
	}

	log.Debug("partial update completed", zap.Int("count", len(keys)))
	return len(keys) == u.batchSize, nil
}

func (u *updater) receipts(receiptDateFrom *lkdr.Date) updateFunc {
	return func(ctx context.Context, log *zap.Logger, offset int) (bool, error) {
		in := &lkdr.ReceiptIn{
			DateFrom: receiptDateFrom,
			OrderBy:  "RECEIVE_DATE:ASC",
			Offset:   offset,
			Limit:    u.batchSize,
		}

		var (
			out *lkdr.ReceiptOut
			err error
		)

		util.WithTimeout(ctx, u.timeout, func(ctx context.Context) { out, err = u.client.Receipt(ctx, in) })

		if err != nil {
			log.Error("failed to get", zap.Error(err))
			return false, errors.New("failed to get")
		}

		brands, err := util.ToViaJSON[[]Brand](out.Brands)
		if err != nil {
			log.Error("brands conversion failed", zap.Error(err))
			return false, errors.New("brand conversion failed")
		}

		if err := u.db.Clauses(util.Upsert("id")).Create(brands).Error; err != nil {
			log.Error("failed to update brands in db", zap.Error(err))
			return false, errors.New("failed to update brands in db")
		}

		receipts, err := util.ToViaJSON[[]Receipt](out.Receipts)
		if err != nil {
			log.Error("receipts conversion failed", zap.Error(err))
			return false, errors.New("receipts conversion failed")
		}

		for i := range receipts {
			receipts[i].UserPhone = u.phone
		}

		if err := u.db.Clauses(util.Upsert("key")).Create(receipts).Error; err != nil {
			log.Error("failed to update receipts in db", zap.Error(err))
			return false, errors.New("failed to update receipts in db")
		}

		return out.HasMore, nil
	}
}
