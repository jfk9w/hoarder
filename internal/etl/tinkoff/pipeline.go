package tinkoff

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-playground/validator"
	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/tinkoff-api"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"gorm.io/gorm"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/etl"
	"github.com/jfk9w/hoarder/internal/util"
	"github.com/jfk9w/hoarder/internal/util/executors"
)

var validate = &based.Lazy[*validator.Validate]{
	Fn: func(ctx context.Context) (*validator.Validate, error) {
		return validator.New(), nil
	},
}

type Builder struct {
	Config Config      `validate:"required"`
	Clock  based.Clock `validate:"required"`
}

func (b Builder) Build(ctx context.Context) (*pipeline, error) {
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
				new(Statement),
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

	return &pipeline{
		clock:           b.Clock,
		clients:         clients,
		db:              db,
		batchSize:       b.Config.BatchSize,
		overlap:         b.Config.Overlap,
		disableReceipts: b.Config.DisableReceipts,
	}, nil
}

type pipeline struct {
	clock           based.Clock
	clients         map[string]map[string]*based.Lazy[Client]
	db              *based.Lazy[*gorm.DB]
	batchSize       int
	overlap         time.Duration
	disableReceipts bool
}

func (p *pipeline) Run(ctx context.Context, log *etl.Logger, username string) (errs error) {
	clients, ok := p.clients[username]
	if !ok {
		return nil
	}

	db, err := p.db.Get(ctx)
	if log.Error(&errs, err, "failed to get db handle") {
		return
	}

	db = db.WithContext(ctx)

	if requestInputFn := etl.GetRequestInputFunc(ctx); requestInputFn != nil {
		ctx = tinkoff.WithAuthorizer(ctx, &authorizer{
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

			client, err := client.Get(ctx)
			if log.Error(&errs, err, "failed to get client for api") {
				return
			}

			var stack util.Stack[update]
			stack.Push(
				&accounts{batchSize: p.batchSize, overlap: p.overlap, phone: phone},
				&investOperationTypes{batchSize: p.batchSize},
				&investAccounts{clock: p.clock, batchSize: p.batchSize, overlap: p.overlap, phone: phone},
			)

			for {
				update, ok := stack.Pop()
				if !ok {
					break
				}

				entity := update.entity()
				log := log.With(slog.String("entity", entity))
				errPrefix := ""
				for _, desc := range update.parent() {
					log = log.With(slog.String(desc.key, desc.value))
					errPrefix = fmt.Sprintf("%s %s: %s", desc.key, desc.value, errPrefix)
				}

				children, err := update.run(ctx, log, client, db)
				for _, err := range multierr.Errors(err) {
					_ = multierr.AppendInto(&errs, errors.Wrap(err, errPrefix+entity))
				}

				stack.Push(children...)
			}

			return
		})
	}

	return executor.Wait()
}
