package jobs

import (
	"go.uber.org/multierr"
)

type Batch[V any] struct {
	Key   string
	Value V
	Size  int
}

func (b Batch[V]) Run(ctx Context, fn func(ctx Context, value V, limit int) (*V, error)) (errs error) {
	value := b.Value
	for {
		ctx := ctx.With(b.Key, value)
		nextValue, err := fn(ctx, value, b.Size)
		if multierr.AppendInto(&errs, err) || nextValue == nil {
			return
		}

		value = *nextValue
	}
}
