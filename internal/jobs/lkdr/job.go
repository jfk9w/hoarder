package lkdr

import (
	"context"
	"hash/fnv"
	"log/slog"
	"math/rand"
	"time"

	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/lkdr-api"
	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/jfk9w/hoarder/internal/captcha"
	"github.com/jfk9w/hoarder/internal/common"
	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/jobs"
	. "github.com/jfk9w/hoarder/internal/jobs/lkdr/internal/entities"
	"github.com/jfk9w/hoarder/internal/jobs/lkdr/internal/loaders"
	"github.com/jfk9w/hoarder/internal/logs"
)

const JobID = "lkdr"

type JobParams struct {
	Config        *Config      `validate:"required"`
	Clock         based.Clock  `validate:"required"`
	Logger        *slog.Logger `validate:"required"`
	ClientFactory ClientFactory
	CaptchaSolver captcha.TokenProvider
}

type Job struct {
	users         map[string]map[string]Client
	batchSize     int
	captchaSolver captcha.TokenProvider
	db            database.DB
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

	storage := &storage{db: db}
	users := make(map[string]map[string]Client)
	for user, credentials := range params.Config.Users {
		phones := make(map[string]Client)
		users[user] = phones
		for _, credential := range credentials {
			deviceID := credential.DeviceID
			if deviceID == "" {
				var err error
				deviceID, err = generateDeviceID(credential.UserAgent, credential.Phone)
				if err != nil {
					return nil, errors.Wrap(err, "generate device ID")
				}
			}

			client, err := lkdr.NewClient(lkdr.ClientParams{
				Phone:        credential.Phone,
				Clock:        params.Clock,
				DeviceID:     deviceID,
				UserAgent:    credential.UserAgent,
				TokenStorage: storage,
			})

			if err != nil {
				return nil, errors.Wrapf(err, "create client for %s/%s", user, credential.Phone)
			}

			phones[credential.Phone] = &boundClient{
				client:  client,
				timeout: params.Config.Timeout,
			}
		}
	}

	return &Job{
		users:         users,
		batchSize:     params.Config.BatchSize,
		captchaSolver: params.CaptchaSolver,
		db:            db,
	}, nil
}

func (j *Job) ID() string {
	return JobID
}

func (j *Job) Run(ctx jobs.Context, now time.Time, userID string) (errs error) {
	phones := j.users[userID]
	if phones == nil {
		return
	}

	ctx = ctx.ApplyAskFn(inAuthorizer)
	for phone, client := range phones {
		ctx := ctx.With("phone", phone)
		err := j.executeLoaders(ctx, now, userID, phone, client)
		_ = multierr.AppendInto(&errs, err)
	}

	return
}

func (j *Job) executeLoaders(ctx jobs.Context, now time.Time, userID, phone string, client Client) (errs error) {
	if err := j.db.WithContext(ctx).
		Upsert(&User{Name: userID, Phone: phone}).
		Error; ctx.Error(&errs, err, "failed to create user in db") {
		return
	}

	var stack common.Stack[loaders.Interface]
	stack.Push(
		loaders.Receipts{Phone: phone, BatchSize: j.batchSize},
		loaders.FiscalData{Phone: phone, BatchSize: j.batchSize},
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
