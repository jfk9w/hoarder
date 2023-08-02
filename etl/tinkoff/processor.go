package tinkoff

import (
	"context"
	"time"

	"github.com/jfk9w/hoarder/etl"

	"github.com/jfk9w-go/based"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/jfk9w/hoarder/database"
	"github.com/jfk9w/hoarder/util"
)

type Processor struct {
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
				new(Card),
				new(Account),
				new(Category),
				new(Location),
				new(LoyaltyBonus),
				new(SpendingCategory),
				new(Brand),
				new(AdditionalInfo),
				new(LoyaltyPayment),
				new(Payment),
				new(Subgroup),
				new(Operation),
				new(ReceiptItem),
				new(Receipt),
			); err != nil {
				return nil, errors.Wrap(err, "migrate db tables")
			}

			return db, nil
		},
	}

	//sessionStorage := &sessionStorage{db: db}
	clients := make(map[string]map[string]*based.Lazy[Client])
	for username, credentials := range cfg.Users {
		clients[username] = make(map[string]*based.Lazy[Client])
		clients := clients[username]
		for _, credential := range credentials {
			credential := credential
			clients[credential.Phone] = &based.Lazy[Client]{
				Fn: func(ctx context.Context) (Client, error) {
					//return tinkoff.ClientBuilder{
					//	Clock: clock,
					//	Credential: tinkoff.Credential{
					//		Phone:    credential.Phone,
					//		Password: credential.Password,
					//	},
					//	SessionStorage: sessionStorage,
					//}.Build(ctx)
					return new(mockClient), nil
				},
			}
		}
	}

	return &Processor{
		clients:   clients,
		db:        db,
		batchSize: cfg.BatchSize,
		overlap:   cfg.Overlap,
	}
}

func (p *Processor) Process(ctx context.Context, stats *etl.Stats, username string) error {
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

		u := &updater{
			client:    client,
			db:        db,
			phone:     phone,
			batchSize: p.batchSize,
			overlap:   p.overlap,
		}

		if err := u.run(ctx, stats.Get(phone, false)); err != nil {
			return errors.Wrapf(err, "for phone %s", phone)
		}
	}

	return nil
}
