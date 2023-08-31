package tinkoff

import (
	"context"
	"fmt"
	"log/slog"
	"time"

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

type Builder struct {
	Config Config      `validate:"required"`
	Clock  based.Clock `validate:"required"`
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
				new(ClientOfferEssenceMccCode),
				new(ClientOfferEssence),
				new(ClientOffer),
			); err != nil {
				return nil, errors.Wrap(err, "migrate db tables")
			}

			return db, nil
		},
	)

	sessionStorage := &sessionStorage{db: db}
	clients := make(map[string]map[string]based.Ref[Client])
	for username, credentials := range b.Config.Users {
		clients[username] = make(map[string]based.Ref[Client])
		clients := clients[username]
		for _, credential := range credentials {
			clients[credential.Phone] = based.Lazy[Client](
				func(ctx context.Context) (Client, error) {
					return tinkoff.ClientBuilder{
						Clock: b.Clock,
						Credential: tinkoff.Credential{
							Phone:    credential.Phone,
							Password: credential.Password,
						},
						SessionStorage: sessionStorage,
					}.Build(ctx)
				},
			)
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
	clients         map[string]map[string]based.Ref[Client]
	db              based.Ref[*gorm.DB]
	batchSize       int
	overlap         time.Duration
	disableReceipts bool
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

			client, err := client(ctx)
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
				updateDesc := entity
				for _, parentDesc := range update.parent() {
					log = log.With(slog.String(parentDesc.key, parentDesc.value))
					updateDesc = fmt.Sprintf("%s: %s %s", updateDesc, parentDesc.key, parentDesc.value)
				}

				children, err := update.run(ctx, log, client, db)
				for _, err := range multierr.Errors(err) {
					_ = multierr.AppendInto(&errs, errors.Wrap(err, updateDesc))
				}

				stack.Push(children...)
			}

			return
		})
	}

	return executor.Wait()
}
