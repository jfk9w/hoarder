package triggers

import (
	"context"
	"log/slog"
	"time"

	"github.com/jfk9w-go/based"
	"github.com/jfk9w/hoarder/internal/jobs"
	"github.com/jfk9w/hoarder/internal/logs"
	"go.uber.org/multierr"
)

type Jobs interface {
	Run(ctx jobs.Context, now time.Time, userID string, jobIDs []string) []jobs.Result
}

type Interface interface {
	ID() string
	Run(ctx Context, jobs Jobs)
}

type Registry struct {
	triggers   []Interface
	goroutines []based.Goroutine
	log        *slog.Logger
}

func NewRegistry(log *slog.Logger) *Registry {
	return &Registry{
		log: log,
	}
}

func (r *Registry) Register(trigger Interface) {
	r.triggers = append(r.triggers, trigger)
}

func (r *Registry) Run(ctx context.Context, job Jobs) {
	for _, trigger := range r.triggers {
		log := r.log.With("trigger", trigger.ID())
		goroutine := based.Go(ctx, func(ctx context.Context) {
			log.Info("trigger started")
			defer log.Info("trigger stopped")
			trigger.Run(NewContext(ctx, log), job)
		})

		go func() {
			if err := goroutine.Join(ctx); err != nil {
				log.Error("panic in trigger", logs.Error(err))
			}
		}()

		r.goroutines = append(r.goroutines, goroutine)
	}
}

func (r *Registry) Close() (errs error) {
	for _, goroutine := range r.goroutines {
		goroutine.Cancel()
		err := goroutine.Join(context.Background())
		_ = multierr.AppendInto(&errs, err)
	}

	return
}
