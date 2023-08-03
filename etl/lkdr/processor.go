package lkdr

import (
	"context"
	"hash/fnv"
	"math/rand"
	"time"

	"github.com/jfk9w/hoarder/etl"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/lkdr-api"

	"github.com/jfk9w/hoarder/captcha"
	"github.com/jfk9w/hoarder/database"
	"github.com/jfk9w/hoarder/util"
)

const Name = "Мои чеки онлайн"

type Processor struct {
	clients       map[string]map[string]*based.Lazy[Client]
	captchaSolver captcha.TokenProvider
	db            *based.Lazy[*gorm.DB]
	batchSize     int
	timeout       time.Duration
}

func NewProcessor(cfg Config, clock based.Clock, captchaSolver captcha.TokenProvider) *Processor {
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
				new(FiscalDataItem),
				new(FiscalData),
			); err != nil {
				return nil, errors.Wrap(err, "migrate db tables")
			}

			return db.Debug(), nil
		},
	}

	tokenStorage := &tokenStorage{db: db}
	clients := make(map[string]map[string]*based.Lazy[Client])
	for username, credentials := range cfg.Users {
		clients[username] = make(map[string]*based.Lazy[Client])
		clients := clients[username]
		for _, credential := range credentials {
			credential := credential
			clients[credential.Phone] = &based.Lazy[Client]{
				Fn: func(ctx context.Context) (Client, error) {
					deviceID, err := generateDeviceID(cfg.UserAgent, credential.Phone)
					if err != nil {
						return nil, errors.Wrap(err, "generate device ID")
					}

					return lkdr.ClientBuilder{
						Phone:        credential.Phone,
						Clock:        clock,
						DeviceID:     deviceID,
						UserAgent:    cfg.UserAgent,
						TokenStorage: tokenStorage,
					}.Build(ctx)
				},
			}
		}
	}

	return &Processor{
		clients:       clients,
		captchaSolver: captchaSolver,
		db:            db,
		batchSize:     cfg.BatchSize,
		timeout:       cfg.Timeout,
	}
}

func generateDeviceID(userAgent, phone string) (string, error) {
	hash := fnv.New64()
	if _, err := hash.Write([]byte(userAgent)); err != nil {
		return "", errors.Wrap(err, "hash user agent")
	}

	if _, err := hash.Write([]byte(phone)); err != nil {
		return "", errors.Wrap(err, "hash phone")
	}

	source := rand.NewSource(int64(hash.Sum64()))

	const symbols = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	var deviceID []byte
	for i := 0; i < 21; i++ {
		deviceID = append(deviceID, symbols[source.Int63()%int64(len(symbols))])
	}

	return string(deviceID), nil
}

func (p *Processor) Name() string {
	return Name
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

	if requestInputFn := etl.GetRequestInputFunc(ctx); p.captchaSolver != nil && requestInputFn != nil {
		ctx = lkdr.WithAuthorizer(ctx, &authorizer{
			captchaSolver:  p.captchaSolver,
			requestInputFn: requestInputFn,
		})
	}

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

		u := &updater{
			client:    client,
			db:        db,
			phone:     phone,
			batchSize: p.batchSize,
			timeout:   p.timeout,
		}

		if err := u.run(ctx); err != nil {
			return errors.Wrapf(err, "for phone %s", phone)
		}
	}

	return nil
}
