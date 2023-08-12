package schedule

import (
	"context"
	"log/slog"
	"time"

	"github.com/jfk9w/hoarder/internal/util/executors"

	"github.com/go-playground/validator/v10"
	"github.com/jfk9w-go/based"
)

type Pipelines interface {
	Run(ctx context.Context, log *slog.Logger, username string) error
}

type Config struct {
	Users    []string      `yaml:"users" doc:"Пользователи, для которых данные нужно синхронизировать в фоновом режиме."`
	Interval time.Duration `yaml:"interval,omitempty" default:"30m" doc:"Интервал синхронизации."`
}

type Builder struct {
	Config    Config       `validate:"required"`
	Pipelines Pipelines    `validate:"required"`
	Log       *slog.Logger `validate:"required"`
}

var validate = &based.Lazy[*validator.Validate]{
	Fn: func(ctx context.Context) (*validator.Validate, error) {
		return validator.New(), nil
	},
}

func (b Builder) Run(ctx context.Context) (context.CancelFunc, error) {
	if validate, err := validate.Get(ctx); err != nil {
		return nil, err
	} else if err := validate.Struct(b); err != nil {
		return nil, err
	}

	cancel := based.GoWithFeedback(ctx, context.WithCancel, func(ctx context.Context) {
		ticker := time.NewTicker(b.Config.Interval)
		defer ticker.Stop()

		for {
			executor := executors.Parallel(b.Log, "username")
			for _, username := range b.Config.Users {
				executor.Run(username, func(log *slog.Logger) error {
					log.Debug("pipelines started")
					if err := b.Pipelines.Run(ctx, log, username); err != nil {
						log.Error("pipelines failed")
						return err
					}

					log.Info("pipelines completed")
					return nil
				})
			}

			_ = executor.Wait()

			select {
			case <-ticker.C:
				continue
			case <-ctx.Done():
				return
			}
		}
	})

	return cancel, nil
}
