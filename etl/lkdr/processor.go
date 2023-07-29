package lkdr

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/lkdr-api"

	"github.com/jfk9w/hoarder/captcha"
	"github.com/jfk9w/hoarder/database"
	"github.com/jfk9w/hoarder/util"
)

type Processor struct {
	clients   map[string]map[string]*based.Lazy[Client]
	db        *based.Lazy[*gorm.DB]
	batchSize int
}

func NewProcessor(cfg Config, clock based.Clock, captchaTokenProvider captcha.TokenProvider) *Processor {
	db := &based.Lazy[*gorm.DB]{
		Fn: func(ctx context.Context) (*gorm.DB, error) {
			db, err := database.Open(cfg.DB)
			if err != nil {
				return nil, errors.Wrap(err, "open db connection")
			}

			if err := db.WithContext(ctx).AutoMigrate(
				new(User),
				new(Tokens),
				new(Brand),
				new(Receipt),
				new(FiscalData),
				new(FiscalDataItem),
			); err != nil {
				return nil, errors.Wrap(err, "migrate db tables")
			}

			return db.Debug(), nil
		},
	}

	tokenStorage := &tokenStorage{db: db}
	clients := make(map[string]map[string]*based.Lazy[Client])
	for username, credentials := range cfg.Users {
		username, credentials := username, credentials
		clients[username] = make(map[string]*based.Lazy[Client])
		clients := clients[username]
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
		clients:   clients,
		db:        db,
		batchSize: cfg.BatchSize,
	}
}

func (p *Processor) Process(ctx context.Context, username string) error {
	clients, ok := p.clients[username]
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

		user := User{
			Name:  username,
			Phone: phone,
		}

		if err := db.Clauses(util.Upsert("phone")).Create(user).Error; err != nil {
			return errors.Wrapf(err, "create user %s:%s in db", username, phone)
		}

		it := &iterator{
			client: client,
			db:     db,
			phone:  phone,
			limit:  p.batchSize,
			init:   isInit(ctx),
		}

		if err := updateData(ctx, it); err != nil {
			return errors.Wrapf(err, "on %s", phone)
		}
	}

	return nil
}

func updateData(ctx context.Context, it *iterator) error {
	var latestReceiptDate sql.NullTime
	if err := it.db.Model(new(Receipt)).
		Select("receive_date").
		Where("user_phone = ?", it.phone).
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

iteratorLoop:
	for _, fn := range []iteratorFunc{
		receipts(receiptDateFrom),
		fiscalData,
	} {
		offset := 0
		for {
			err := fn(ctx, it, offset)
			switch {
			case errors.Is(err, errStopIteration):
				continue iteratorLoop
			case err != nil:
				err = errors.Wrapf(err, "on offset %d", offset)
				log.Println(err)
				continue iteratorLoop
			}

			if it.init {
				break
			}

			offset += it.limit
		}
	}

	return nil
}

func fiscalData(ctx context.Context, it *iterator, offset int) error {
	var receiptKeys []string
	if err := it.db.Model(new(Receipt)).
		Select("receipts.key").
		Joins("left join fiscal_data on receipts.key = fiscal_data.receipt_key").
		Where("receipts.user_phone = ? and fiscal_data.receipt_key is null", it.phone).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "receive_date"}}).
		Offset(offset).
		Limit(it.limit).
		Scan(&receiptKeys).
		Error; err != nil {
		return errors.Wrap(err, "select receipt keys w/o fiscal data")
	}

	for _, key := range receiptKeys {
		fiscalDataOut, err := getFiscalData(ctx, it.client, key)
		if err != nil {
			err = errors.Wrapf(err, "on key %s", key)
			log.Println(err)
			continue
		}

		fiscalData, err := util.ToViaJSON[FiscalData](fiscalDataOut)
		if err != nil {
			return errors.Wrapf(err, "convert fiscal data %s to entity", key)
		}

		fiscalData.Receipt.Key = key
		for i := range fiscalData.Items {
			fiscalData.Items[i].ReceiptKey = key
			fiscalData.Items[i].Position = i + 1
		}

		if err := it.db.Clauses(util.Upsert("receipt_key")).Create(&fiscalData).Error; err != nil {
			return errors.Wrapf(err, "upsert fiscal data %s", key)
		}
	}

	if len(receiptKeys) < it.limit {
		return errStopIteration
	}

	return nil
}

func getFiscalData(ctx context.Context, client Client, key string) (*lkdr.FiscalDataOut, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return client.FiscalData(ctx, &lkdr.FiscalDataIn{Key: key})
}

func receipts(receiptDateFrom *lkdr.Date) iteratorFunc {
	return func(ctx context.Context, it *iterator, offset int) error {
		receiptIn := &lkdr.ReceiptIn{
			DateFrom: receiptDateFrom,
			OrderBy:  "RECEIVE_DATE:ASC",
			Offset:   offset,
			Limit:    it.limit,
		}

		receiptOut, err := it.client.Receipt(ctx, receiptIn)
		if err != nil {
			return errors.Wrap(err, "get receipts")
		}

		brands, err := util.ToViaJSON[[]Brand](receiptOut.Brands)
		if err != nil {
			return errors.Wrap(err, "convert brands to entities")
		}

		if err := it.db.Clauses(util.Upsert("id")).Create(brands).Error; err != nil {
			return errors.Wrap(err, "upsert brands")
		}

		receipts, err := util.ToViaJSON[[]Receipt](receiptOut.Receipts)
		if err != nil {
			return errors.Wrap(err, "convert receipts to entities")
		}

		for i := range receipts {
			receipt := &receipts[i]
			receipt.UserPhone = it.phone
		}

		if err := it.db.Clauses(util.Upsert("key")).Create(receipts).Error; err != nil {
			return errors.Wrap(err, "upsert receipts")
		}

		if !receiptOut.HasMore {
			return errStopIteration
		}

		return nil
	}
}

var errStopIteration = errors.New("stop iteration")

type iteratorFunc func(ctx context.Context, it *iterator, offset int) error

type iterator struct {
	client Client
	db     *gorm.DB
	phone  string
	limit  int
	init   bool
}
