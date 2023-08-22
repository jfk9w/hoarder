package schedule

import (
	"context"
	"log/slog"
	"time"

	"github.com/jfk9w-go/based"
	"github.com/jfk9w/hoarder/internal/util/executors"
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

func (b Builder) Run(ctx context.Context) (*handler, error) {
	if err := based.Validate.Struct(b); err != nil {
		return nil, err
	}

	h := &handler{
		interval:  b.Config.Interval,
		users:     b.Config.Users,
		pipelines: b.Pipelines,
		log:       b.Log,
	}

	h.looper = based.Go(ctx, h.loop)

	return h, nil
}

type handler struct {
	interval  time.Duration
	users     []string
	pipelines Pipelines
	log       *slog.Logger
	looper    based.Goroutine
}

func (h *handler) loop(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		executor := executors.Parallel(h.log, "username")
		for _, username := range h.users {
			executor.Run(username, func(log *slog.Logger) error {
				log.Debug("pipelines started")
				if err := h.pipelines.Run(ctx, log, username); err != nil {
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
}

func (h *handler) Stop() {
	h.looper.Cancel()
	_ = h.looper.Join(context.Background())
}
