package lkdr

import (
	"context"
	"hash/fnv"
	"math/rand"
	"sync"
	"time"

	"github.com/go-playground/validator"
	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/lkdr-api"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/jfk9w/hoarder/internal/captcha"
	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/etl"
)

const Name = "lkdr"

var validate = &based.Lazy[*validator.Validate]{
	Fn: func(ctx context.Context) (*validator.Validate, error) {
		return validator.New(), nil
	},
}

type Builder struct {
	Config Config      `validate:"required"`
	Clock  based.Clock `validate:"required"`
	Log    *zap.Logger `validate:"required"`

	CaptchaSolver captcha.TokenProvider
}

func (b Builder) Build(ctx context.Context) (*Processor, error) {
	if validate, err := validate.Get(ctx); err != nil {
		return nil, err
	} else if err := validate.Struct(b); err != nil {
		return nil, err
	}

	db := &based.Lazy[*gorm.DB]{
		Fn: func(ctx context.Context) (*gorm.DB, error) {
			db, err := database.Open(b.Config.DB)
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
	}

	tokenStorage := &tokenStorage{db: db}
	clients := make(map[string]map[string]*based.Lazy[Client])
	for username, credentials := range b.Config.Users {
		clients[username] = make(map[string]*based.Lazy[Client])
		clients := clients[username]
		for _, credential := range credentials {
			credential := credential
			clients[credential.Phone] = &based.Lazy[Client]{
				Fn: func(ctx context.Context) (Client, error) {
					deviceID, err := generateDeviceID(b.Config.UserAgent, credential.Phone)
					if err != nil {
						return nil, errors.Wrap(err, "generate device ID")
					}

					return lkdr.ClientBuilder{
						Phone:        credential.Phone,
						Clock:        b.Clock,
						DeviceID:     deviceID,
						UserAgent:    b.Config.UserAgent,
						TokenStorage: tokenStorage,
					}.Build(ctx)
				},
			}
		}
	}

	return &Processor{
		log:           b.Log,
		clients:       clients,
		captchaSolver: b.CaptchaSolver,
		db:            db,
		batchSize:     b.Config.BatchSize,
		timeout:       b.Config.Timeout,
	}, nil
}

type Processor struct {
	log           *zap.Logger
	clients       map[string]map[string]*based.Lazy[Client]
	captchaSolver captcha.TokenProvider
	db            *based.Lazy[*gorm.DB]
	batchSize     int
	timeout       time.Duration
}

func (p *Processor) Name() string {
	return Name
}

func (p *Processor) Process(ctx context.Context, username string) (errs error) {
	clients, ok := p.clients[username]
	if !ok {
		return nil
	}

	log := p.log.With(zap.String("username", username))

	db, err := p.db.Get(ctx)
	if err != nil {
		log.Error("failed to get db handle", zap.Error(err))
		return errors.New("failed to get db handle")
	}

	db = db.WithContext(ctx)

	if requestInputFn := etl.GetRequestInputFunc(ctx); p.captchaSolver != nil && requestInputFn != nil {
		ctx = lkdr.WithAuthorizer(ctx, &authorizer{
			captchaSolver:  p.captchaSolver,
			requestInputFn: requestInputFn,
		})
	}

	var (
		errc = make(chan error, len(clients))
		work sync.WaitGroup
	)

	for phone, client := range clients {
		work.Add(1)
		go func(phone string, lazyClient *based.Lazy[Client]) {
			defer work.Done()
			log := log.With(zap.String("phone", phone))
			errFn := func(err error, msg string) error {
				if err == nil {
					return nil
				}

				log.Error(msg, zap.Error(err))
				return errors.Errorf("%s: %s", phone, msg)
			}

			client, err := lazyClient.Get(ctx)
			if err := errFn(err, "failed to get client"); err != nil {
				errc <- err
				return
			}

			user := User{
				Name:  username,
				Phone: phone,
			}

			err = db.Clauses(etl.Upsert("phone")).Create(user).Error
			if err := errFn(err, "failed to create user in db"); err != nil {
				errc <- err
				return
			}

			u := &updater{
				client:    client,
				db:        db,
				phone:     phone,
				batchSize: p.batchSize,
				timeout:   p.timeout,
			}

			for _, err := range multierr.Errors(u.run(ctx, log)) {
				errc <- errors.Wrap(err, phone)
			}
		}(phone, client)
	}

	work.Wait()
	close(errc)

	if ctx.Err() != nil {
		return ctx.Err()
	}

	for _, err := range multierr.Errors(err) {
		errs = multierr.Append(errs, err)
	}

	return
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
