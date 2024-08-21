package tinkoff

import (
	"context"
	"log/slog"
	"time"

	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/tinkoff-api"
	"github.com/pkg/errors"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	"go.uber.org/multierr"

	"github.com/jfk9w/hoarder/internal/common"
	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/firefly"
	"github.com/jfk9w/hoarder/internal/jobs"
	. "github.com/jfk9w/hoarder/internal/jobs/tinkoff/internal/entities"
	"github.com/jfk9w/hoarder/internal/jobs/tinkoff/internal/loaders"
	fireflySync "github.com/jfk9w/hoarder/internal/jobs/tinkoff/internal/sync/firefly"
	"github.com/jfk9w/hoarder/internal/logs"
)

const JobID = "tinkoff"

type JobParams struct {
	Clock         based.Clock  `validate:"required"`
	Logger        *slog.Logger `validate:"required"`
	Config        Config       `validate:"required"`
	ClientFactory ClientFactory
	Firefly       firefly.Invoker
}

type Job struct {
	users        map[string]map[string]pingingClient
	batchSize    int
	overlap      time.Duration
	withReceipts bool
	db           database.DB
	firefly      firefly.Invoker
}

func NewJob(ctx context.Context, params JobParams) (*Job, error) {
	if err := based.Validate(params); err != nil {
		return nil, err
	}

	if params.ClientFactory == nil {
		params.ClientFactory = defaultClientFactory
	}

	db, err := database.Open(ctx, database.Params{
		Clock:    params.Clock,
		Logger:   params.Logger.With(logs.Database(JobID)),
		Config:   params.Config.Database,
		Entities: entities,
	})

	if err != nil {
		return nil, err
	}

	authFlow := tinkoff.ApiAuthFlow
	if cfg := params.Config.Selenium; cfg != nil && cfg.Enabled {
		caps := selenium.Capabilities{"browserName": cfg.Browser}
		caps.AddChrome(chrome.Capabilities{Args: cfg.Args, Path: cfg.Binary})
		authFlow = &tinkoff.SeleniumAuthFlow{
			Capabilities: caps,
			URLPrefix:    cfg.URLPrefix,
		}
	}

	storage := &storage{db: db}
	users := make(map[string]map[string]pingingClient)
	for user, credentials := range params.Config.Users {
		phones := make(map[string]pingingClient)
		users[user] = phones
		for _, credential := range credentials {
			client, err := params.ClientFactory(tinkoff.ClientParams{
				Clock: params.Clock,
				Credential: tinkoff.Credential{
					Phone:    credential.Phone,
					Password: credential.Password,
				},
				SessionStorage: storage,
				AuthFlow:       authFlow,
			})

			if err != nil {
				return nil, errors.Wrapf(err, "create client for %s/%s", user, credential.Phone)
			}

			phones[credential.Phone] = pingingClient{
				Client: client,
				pinger: based.Go(context.Background(), client.Ping),
			}
		}
	}

	return &Job{
		users:        users,
		batchSize:    params.Config.BatchSize,
		overlap:      params.Config.Overlap,
		withReceipts: params.Config.WithReceipts,
		db:           db,
		firefly:      params.Firefly,
	}, nil
}

func (j *Job) Close() (errs error) {
	for _, phones := range j.users {
		for phone, client := range phones {
			_ = multierr.AppendInto(&errs, errors.Wrap(client.Close(), phone))
		}
	}

	return
}

func (j *Job) Info() jobs.Info {
	return jobs.Info{
		ID:          JobID,
		Description: "Загрузка счетов, операций и пр. из Т-Банка и Т-Инвестиций",
	}
}

func (j *Job) Run(ctx jobs.Context, now time.Time, userID string) (errs error) {
	phones := j.users[userID]
	if phones == nil {
		return jobs.ErrJobUnconfigured
	}

	ctx = ctx.ApplyAskFn(withAuthorizer)
	for phone, client := range phones {
		ctx := ctx.With("phone", phone)
		err := j.executeLoaders(ctx, now, userID, phone, client)
		_ = multierr.AppendInto(&errs, err)
	}

	if err := j.executeFireflySync(ctx, userID); err != nil {
		_ = multierr.AppendInto(&errs, err)
	}

	return
}

func (j *Job) executeLoaders(ctx jobs.Context, now time.Time, userID string, phone string, client Client) (errs error) {
	if err := j.db.WithContext(ctx).
		Upsert(&User{Name: userID, Phone: phone}).
		Error; ctx.Error(&errs, err, "failed to create user in db") {
		return
	}

	var stack common.Stack[loaders.Interface]
	stack.Push(
		loaders.ClientOffers{Phone: phone, BatchSize: j.batchSize},
		loaders.Accounts{Phone: phone, BatchSize: j.batchSize, Overlap: j.overlap, WithReceipts: j.withReceipts},
		loaders.InvestOperationTypes{BatchSize: j.batchSize},
		loaders.InvestAccounts{Phone: phone, BatchSize: j.batchSize, Overlap: j.overlap, Now: now},
	)

	for {
		loader, ok := stack.Pop()
		if !ok {
			break
		}

		ctx := ctx.With("entity", loader.TableName())
		loaders, err := loader.Load(ctx, client, j.db)
		if !multierr.AppendInto(&errs, err) {
			stack.Push(loaders...)
		}
	}

	return
}

func (j *Job) executeFireflySync(ctx jobs.Context, userID string) (errs error) {
	if j.firefly == nil {
		return
	}

	var phones []string
	for phone := range j.users[userID] {
		phones = append(phones, phone)
	}

	var stack common.Stack[fireflySync.Interface]
	stack.Push(fireflySync.All{Phones: phones, BatchSize: j.batchSize})

	for {
		sync, ok := stack.Pop()
		if !ok {
			break
		}

		ctx := ctx.With("entity", sync.TableName())
		syncs, err := sync.Sync(ctx, j.db, j.firefly)
		if !multierr.AppendInto(&errs, err) {
			stack.Push(syncs...)
		}
	}

	return
}
