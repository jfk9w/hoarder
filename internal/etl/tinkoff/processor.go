package tinkoff

import (
	"context"
	"time"

	"github.com/go-playground/validator"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/jfk9w-go/tinkoff-api"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/etl"
	"github.com/jfk9w/hoarder/internal/util"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/jfk9w-go/based"
)

const Name = "tinkoff"

var validate = &based.Lazy[*validator.Validate]{
	Fn: func(ctx context.Context) (*validator.Validate, error) {
		return validator.New(), nil
	},
}

type Builder struct {
	Config Config      `validate:"required"`
	Clock  based.Clock `validate:"required"`
	Log    *zap.Logger `validate:"required"`
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
				new(Session),
				new(Currency),
				new(Account),
				new(Card),
				new(Category),
				new(SpendingCategory),
				new(Brand),
				new(Subgroup),
				new(Operation),
				new(Location),
				new(LoyaltyBonus),
				new(AdditionalInfo),
				new(LoyaltyPayment),
				new(Payment),
				new(Receipt),
				new(ReceiptItem),
				new(InvestOperationType),
				new(InvestAccount),
				new(InvestOperation),
				new(Trade),
				new(InvestChildOperation),
			); err != nil {
				return nil, errors.Wrap(err, "migrate db tables")
			}

			return db, nil
		},
	}

	sessionStorage := &sessionStorage{db: db}
	clients := make(map[string]map[string]*based.Lazy[Client])
	for username, credentials := range b.Config.Users {
		clients[username] = make(map[string]*based.Lazy[Client])
		clients := clients[username]
		for _, credential := range credentials {
			credential := credential
			clients[credential.Phone] = &based.Lazy[Client]{
				Fn: func(ctx context.Context) (Client, error) {
					return tinkoff.ClientBuilder{
						Clock: b.Clock,
						Credential: tinkoff.Credential{
							Phone:    credential.Phone,
							Password: credential.Password,
						},
						SessionStorage: sessionStorage,
					}.Build(ctx)
				},
			}
		}
	}

	return &Processor{
		log:       b.Log,
		clock:     b.Clock,
		clients:   clients,
		db:        db,
		batchSize: b.Config.BatchSize,
		overlap:   b.Config.Overlap,
	}, nil
}

type Processor struct {
	log       *zap.Logger
	clock     based.Clock
	clients   map[string]map[string]*based.Lazy[Client]
	db        *based.Lazy[*gorm.DB]
	batchSize int
	overlap   time.Duration
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

	if requestInputFn := etl.GetRequestInputFunc(ctx); requestInputFn != nil {
		ctx = tinkoff.WithAuthorizer(ctx, &authorizer{
			requestInputFn: requestInputFn,
		})
	}

	for phone, client := range clients {
		log := log.With(zap.String("phone", phone))
		errFn := func(err error, msg string) error {
			if err == nil {
				return nil
			}

			log.Error(msg, zap.Error(err))
			return errors.Errorf("%s: %s", phone, msg)
		}

		client, err := client.Get(ctx)
		if multierr.AppendInto(&errs, errFn(err, "failed to get client")) {
			continue
		}

		user := User{
			Name:  username,
			Phone: phone,
		}

		err = db.Clauses(util.Upsert("phone")).Create(user).Error
		if multierr.AppendInto(&errs, errFn(err, "failed to create user in db")) {
			continue
		}

		u := &updater{
			clock:     p.clock,
			client:    client,
			db:        db,
			phone:     phone,
			batchSize: p.batchSize,
			overlap:   p.overlap,
		}

		for _, err := range multierr.Errors(u.run(ctx, log)) {
			errs = multierr.Append(errs, errors.Wrap(err, phone))
		}
	}

	return errs
}
