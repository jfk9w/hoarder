package tinkoff

import (
	"context"
	"sync"
	"time"

	"github.com/go-playground/validator"
	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/tinkoff-api"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/etl"
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
			db, err := database.Open(b.Clock, b.Config.DB)
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
		log:             b.Log,
		clock:           b.Clock,
		clients:         clients,
		db:              db,
		batchSize:       b.Config.BatchSize,
		overlap:         b.Config.Overlap,
		disableReceipts: b.Config.DisableReceipts,
	}, nil
}

type Processor struct {
	log             *zap.Logger
	clock           based.Clock
	clients         map[string]map[string]*based.Lazy[Client]
	db              *based.Lazy[*gorm.DB]
	batchSize       int
	overlap         time.Duration
	disableReceipts bool
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
				clock:           p.clock,
				client:          client,
				db:              db,
				phone:           phone,
				batchSize:       p.batchSize,
				overlap:         p.overlap,
				disableReceipts: p.disableReceipts,
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

	for err := range errc {
		errs = multierr.Append(errs, err)
	}

	return
}
