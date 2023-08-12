package executors

import (
	"log/slog"
	"sync"

	"github.com/pkg/errors"

	"go.uber.org/multierr"
)

type parallel[L Logger[L]] struct {
	log   L
	split string
	err   error
	mu    sync.Mutex
	wg    sync.WaitGroup
}

func Parallel[L Logger[L]](log L, split string) *parallel[L] {
	return &parallel[L]{
		log:   log,
		split: split,
	}
}

func (e *parallel[L]) Run(key string, fn func(log L) error) {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		log := e.log.With(slog.String(e.split, key))
		err := fn(log)
		e.mu.Lock()
		defer e.mu.Unlock()
		for _, err := range multierr.Errors(err) {
			_ = multierr.AppendInto(&e.err, errors.Wrap(err, key))
		}
	}()
}

func (e *parallel[L]) Wait() error {
	e.wg.Wait()
	return e.err
}
