package etl

import (
	"context"
	"log/slog"

	"github.com/jfk9w/hoarder/internal/util"
	"github.com/jfk9w/hoarder/internal/util/executors"

	"github.com/pkg/errors"
)

type Pipeline interface {
	Run(ctx context.Context, log *Logger, username string) error
}

type Registry struct {
	pipelines map[string]Pipeline
	mutex     util.MultiMutex[string]
}

func (r *Registry) Register(id string, pipeline Pipeline) {
	if r.pipelines == nil {
		r.pipelines = make(map[string]Pipeline)
	}

	r.pipelines[id] = pipeline
}

func (r *Registry) Run(ctx context.Context, log *slog.Logger, username string) error {
	unlock, ok := r.mutex.TryLock(username)
	if !ok {
		return errors.New("already running")
	}

	defer unlock()

	executor := executors.Parallel(log, "pipeline")
	for id, pipeline := range r.pipelines {
		executor.Run(id, func(log *slog.Logger) error {
			log.Debug("pipeline started")
			defer log.Debug("pipeline completed")
			return pipeline.Run(ctx, &Logger{Logger: log}, username)
		})
	}

	return executor.Wait()
}
