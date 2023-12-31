package lkdr

import (
	"context"
	"hash/fnv"
	"log/slog"
	"math/rand"

	"github.com/jfk9w/hoarder/internal/util/executors"

	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/lkdr-api"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"gorm.io/gorm"

	"github.com/jfk9w/hoarder/internal/captcha"
	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/etl"
)

type Builder struct {
	Config Config      `validate:"required"`
	Clock  based.Clock `validate:"required"`

	CaptchaSolver captcha.TokenProvider
}

func (b Builder) Build() (*pipeline, error) {
	if err := based.Validate.Struct(b); err != nil {
		return nil, err
	}

	db := based.Lazy[*gorm.DB](
		func(ctx context.Context) (*gorm.DB, error) {
			db, err := database.Open(b.Clock, b.Config.DB)
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

			return db, nil
		},
	)

	tokenStorage := &tokenStorage{db: db}
	clients := make(map[string]map[string]based.Ref[Client])
	for username, credentials := range b.Config.Users {
		clients[username] = make(map[string]based.Ref[Client])
		clients := clients[username]
		for _, credential := range credentials {
			clients[credential.Phone] = based.Lazy[Client](
				func(ctx context.Context) (Client, error) {
					deviceID := credential.DeviceID
					if deviceID == "" {
						var err error
						deviceID, err = generateDeviceID(credential.UserAgent, credential.Phone)
						if err != nil {
							return nil, errors.Wrap(err, "generate device ID")
						}
					}

					client, err := lkdr.ClientBuilder{
						Phone:        credential.Phone,
						Clock:        b.Clock,
						DeviceID:     deviceID,
						UserAgent:    credential.UserAgent,
						TokenStorage: tokenStorage,
					}.Build(ctx)
					if err != nil {
						return nil, errors.Wrap(err, "create client")
					}

					return &boundClient{
						client:  client,
						timeout: b.Config.Timeout,
					}, nil
				},
			)
		}
	}

	return &pipeline{
		clients:       clients,
		captchaSolver: b.CaptchaSolver,
		db:            db,
		batchSize:     b.Config.BatchSize,
	}, nil
}

type pipeline struct {
	clients       map[string]map[string]based.Ref[Client]
	captchaSolver captcha.TokenProvider
	db            based.Ref[*gorm.DB]
	batchSize     int
}

func (p *pipeline) Run(ctx context.Context, log *etl.Logger, username string) (errs error) {
	clients, ok := p.clients[username]
	if !ok {
		return nil
	}

	db, err := p.db(ctx)
	if log.Error(&errs, err, "failed to get db handle") {
		return
	}

	db = db.WithContext(ctx)

	if requestInputFn := etl.GetRequestInputFunc(ctx); p.captchaSolver != nil && requestInputFn != nil {
		ctx = lkdr.WithAuthorizer(ctx, &authorizer{
			captchaSolver:  p.captchaSolver,
			requestInputFn: requestInputFn,
		})
	}

	executor := executors.Parallel(log, "phone")
	for phone, client := range clients {
		executor.Run(phone, func(log *etl.Logger) (errs error) {
			if err := db.Clauses(etl.Upsert("phone")).
				Create(&User{Name: username, Phone: phone}).
				Error; log.Error(&errs, err, "failed to create user in db") {
				return
			}

			client, err := client(ctx)
			if log.Error(&errs, err, "failed to get client for api") {
				return
			}

			for _, update := range []update{
				&receipts{phone: phone, batchSize: p.batchSize},
				&fiscalData{phone: phone, batchSize: p.batchSize},
			} {
				entity := update.entity()
				log := log.With(slog.String("entity", entity))
				log.Debug("update started")
				if err := update.run(ctx, log, client, db); err != nil {
					for _, err := range multierr.Errors(err) {
						_ = multierr.AppendInto(&errs, errors.Wrap(err, entity))
					}

					log.Warn("update failed")
					return
				}

				log.Debug("update completed")
			}

			return
		})
	}

	return executor.Wait()
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
