package schedule

import (
	"context"
	"time"

	"go.uber.org/multierr"

	"github.com/go-playground/validator"
	"github.com/jfk9w-go/based"
	"go.uber.org/zap"
)

type Processor interface {
	Process(ctx context.Context, username string) error
}

type Config struct {
	Users    []string      `yaml:"users" doc:"Пользователи, для которых данные нужно синхронизировать в фоновом режиме."`
	Interval time.Duration `yaml:"interval,omitempty" default:"30m" doc:"Интервал синхронизации."`
}

type Builder struct {
	Config    Config      `validate:"required"`
	Processor Processor   `validate:"required"`
	Log       *zap.Logger `validate:"required"`
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
			for _, username := range b.Config.Users {
				if err := b.Processor.Process(ctx, username); err != nil {
					b.Log.Error("process failed", zap.Errors("errors", multierr.Errors(err)))
				} else {
					b.Log.Info("process completed")
				}

				select {
				case <-ticker.C:
					continue
				case <-ctx.Done():
					return
				}
			}
		}
	})

	return cancel, nil
}
