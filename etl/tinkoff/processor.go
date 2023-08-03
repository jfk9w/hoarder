package tinkoff

import (
	"context"
	"time"

	"github.com/jfk9w-go/tinkoff-api"
	"github.com/jfk9w/hoarder/etl"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/jfk9w-go/based"

	"github.com/jfk9w/hoarder/database"
	"github.com/jfk9w/hoarder/util"
)

const Name = "tinkoff"

type Processor struct {
	clock     based.Clock
	clients   map[string]map[string]*based.Lazy[Client]
	db        *based.Lazy[*gorm.DB]
	batchSize int
	overlap   time.Duration
}

func NewProcessor(cfg Config, clock based.Clock) *Processor {
	db := &based.Lazy[*gorm.DB]{
		Fn: func(ctx context.Context) (*gorm.DB, error) {
			db, err := database.Open(cfg.DB)
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
	for username, credentials := range cfg.Users {
		clients[username] = make(map[string]*based.Lazy[Client])
		clients := clients[username]
		for _, credential := range credentials {
			credential := credential
			clients[credential.Phone] = &based.Lazy[Client]{
				Fn: func(ctx context.Context) (Client, error) {
					return tinkoff.ClientBuilder{
						Clock: clock,
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
		clock:     clock,
		clients:   clients,
		db:        db,
		batchSize: cfg.BatchSize,
		overlap:   cfg.Overlap,
	}
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

	if requestInputFn := etl.GetRequestInputFunc(ctx); requestInputFn != nil {
		ctx = tinkoff.WithAuthorizer(ctx, &authorizer{
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
			clock:     p.clock,
			client:    client,
			db:        db,
			phone:     phone,
			batchSize: p.batchSize,
			overlap:   p.overlap,
		}

		if err := u.run(ctx); err != nil {
			return errors.Wrapf(err, "for phone %s", phone)
		}
	}

	return nil
}
