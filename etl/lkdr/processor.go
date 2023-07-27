package lkdr

import (
	"context"
	"database/sql"

	"gorm.io/gorm/clause"

	"github.com/AlekSi/pointer"
	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/lkdr-api"
	"github.com/jfk9w/hoarder/database"
	"github.com/jfk9w/hoarder/util"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type Processor struct {
	clients map[string]map[string]*based.Lazy[Client]
	db      *based.Lazy[*gorm.DB]
}

func NewProcessor(cfg Config, clock based.Clock, rucaptchaClient RucaptchaClient) *Processor {
	db := &based.Lazy[*gorm.DB]{
		Fn: func(ctx context.Context) (*gorm.DB, error) {
			db, err := database.Open(cfg.DB)
			if err != nil {
				return nil, errors.Wrap(err, "open db connection")
			}

			if err := db.WithContext(ctx).AutoMigrate(
				new(Tokens),
				new(Brand),
				new(Receipt),
				new(FiscalData),
				new(FiscalDataItem),
			); err != nil {
				return nil, errors.Wrap(err, "migrate db tables")
			}

			return db, nil
		},
	}

	var (
		tokenStorage         = &tokenStorage{db: db}
		captchaTokenProvider lkdr.CaptchaTokenProvider
	)

	if rucaptchaClient != nil {
		captchaTokenProvider = &rucaptchaTokenProvider{client: rucaptchaClient}
	}

	clients := make(map[string]map[string]*based.Lazy[Client])
	for tenant, credentials := range cfg.Tenants {
		tenant, credentials := tenant, credentials
		clients[tenant] = make(map[string]*based.Lazy[Client])
		clients := clients[tenant]
		for _, credential := range credentials {
			credential := credential
			clients[credential.Phone] = &based.Lazy[Client]{
				Fn: func(ctx context.Context) (Client, error) {
					return lkdr.ClientBuilder{
						Phone:                credential.Phone,
						Clock:                clock,
						DeviceID:             cfg.DeviceID,
						UserAgent:            cfg.UserAgent,
						TokenStorage:         tokenStorage,
						CaptchaTokenProvider: captchaTokenProvider,
						ConfirmationProvider: stdinConfirmationProvider{},
					}.Build(ctx)
				},
			}
		}
	}

	return &Processor{
		clients: clients,
		db:      db,
	}
}

func (p *Processor) Process(ctx context.Context, tenant string) error {
	clients, ok := p.clients[tenant]
	if !ok {
		return nil
	}

	db, err := p.db.Get(ctx)
	if err != nil {
		return errors.Wrap(err, "get db handle")
	}

	db = db.WithContext(ctx)

	for phone, client := range clients {
		client, err := client.Get(ctx)
		if err != nil {
			return errors.Wrapf(err, "get client for %s", phone)
		}

		if err := updateData(ctx, tenant, phone, client, db); err != nil {
			return errors.Wrapf(err, "update data for %s", phone)
		}
	}

	return nil
}

func updateData(ctx context.Context, tenant, phone string, client Client, db *gorm.DB) error {
	if err := updateReceipts(ctx, tenant, phone, client, db); err != nil {
		return errors.Wrap(err, "update receipts")
	}

	if err := updateFiscalData(ctx, phone, client, db); err != nil {
		return errors.Wrap(err, "update fiscal data")
	}

	return nil
}

func updateFiscalData(ctx context.Context, phone string, client Client, db *gorm.DB) error {
	var receiptKeys []string
	if err := db.Model(new(Receipt)).
		Select("receipts.key").
		Joins("left join fiscal_data on receipts.key = fiscal_data.receipt_key").
		Where("receipts.phone = ? and fiscal_data.receipt_key is null", phone).
		Scan(&receiptKeys).
		Error; err != nil {
		return errors.Wrap(err, "select receipt keys w/o fiscal data")
	}

	for _, key := range receiptKeys {
		fiscalDataIn := &lkdr.FiscalDataIn{
			Key: key,
		}

		fiscalDataOut, err := client.FiscalData(ctx, fiscalDataIn)
		if err != nil {
			// TODO error handling
			//return errors.Wrapf(err, "get fiscal data %s", key)
			continue
		}

		fiscalData, err := util.ToViaJSON[FiscalData](fiscalDataOut)
		if err != nil {
			return errors.Wrapf(err, "convert fiscal data %s to entity", key)
		}

		fiscalData.ReceiptKey = key
		for i := range fiscalData.Items {
			fiscalData.Items[i].ReceiptKey = key
		}

		if err := db.Create(fiscalData).Error; err != nil {
			return errors.Wrapf(err, "upsert fiscal data %s", key)
		}
	}

	return nil
}

const receiptLimit = 1000

func updateReceipts(ctx context.Context, tenant, phone string, client Client, db *gorm.DB) error {
	var latestReceiptDate sql.NullTime
	if err := db.Model(new(Receipt)).
		Select("receive_date").
		Where("phone = ?", phone).
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

	var (
		hasMore = true
		offset  = 0
	)

	for hasMore {
		receiptIn := &lkdr.ReceiptIn{
			DateFrom: receiptDateFrom,
			OrderBy:  "RECEIVE_DATE:ASC",
			Offset:   offset,
			Limit:    receiptLimit,
		}

		receiptOut, err := client.Receipt(ctx, receiptIn)
		if err != nil {
			return errors.Wrap(err, "get receipts")
		}

		brands, err := util.ToViaJSON[[]Brand](receiptOut.Brands)
		if err != nil {
			return errors.Wrap(err, "convert brands to entities")
		}

		if err := db.Clauses(util.Upsert("id")).Create(brands).Error; err != nil {
			return errors.Wrap(err, "upsert brands")
		}

		receipts, err := util.ToViaJSON[[]Receipt](receiptOut.Receipts)
		if err != nil {
			return errors.Wrap(err, "convert receipts to entities")
		}

		for i := range receipts {
			receipt := &receipts[i]
			receipt.Tenant = tenant
			receipt.Phone = phone
		}

		if err := db.Clauses(util.Upsert("key")).Create(receipts).Error; err != nil {
			return errors.Wrap(err, "upsert receipts")
		}

		hasMore = receiptOut.HasMore
		offset += receiptLimit
	}

	return nil
}
