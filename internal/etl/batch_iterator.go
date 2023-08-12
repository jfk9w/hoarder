package etl

import (
	"log/slog"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

type BatchIterator[V any] struct {
	BatchSize int
	Log       *Logger
	Key       string
	Value     V
}

func (it *BatchIterator[V]) Run(fn func(log *Logger, value V, batchSize int) (*V, error)) (errs error) {
	value := it.Value
	for {
		log := it.Log.With(slog.Any(it.Key, value))
		nextValue, err := fn(log, value, it.BatchSize)
		if err != nil {
			for _, err := range multierr.Errors(err) {
				_ = multierr.AppendInto(&errs, errors.Wrapf(err, "%v", value))
			}

			return
		}

		if nextValue == nil {
			return
		}

		value = *nextValue
	}
}
