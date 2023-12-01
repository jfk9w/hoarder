package based

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
)

// Ref references a value which may require calculation.
type Ref[T any] interface {
	Get(ctx context.Context) (T, error)
}

type FuncRef[T any] func(ctx context.Context) (T, error)

func (r FuncRef[T]) Get(ctx context.Context) (T, error) {
	return r(ctx)
}

type safeRef[T any] struct {
	ref Ref[T]
}

func (r *safeRef[T]) Get(ctx context.Context) (result T, err error) {
	defer func() {
		if r := recover(); r != nil {
			if r, ok := r.(error); ok {
				err = r
				return
			}

			err = errors.Errorf("panic: %v", r)
		}
	}()

	if r.ref == nil {
		err = errors.New("ref is nil")
		return
	}

	result, err = r.ref.Get(ctx)
	return
}

// SafeRef calls Ref with panic handling and other safeguards.
func SafeRef[T any](ref Ref[T]) Ref[T] {
	if ref, ok := ref.(*safeRef[T]); ok {
		return ref
	}

	return &safeRef[T]{
		ref: ref,
	}
}

func SafeFuncRef[T any](ref Ref[T]) Ref[T] {
	return SafeRef[T](ref)
}

type lazyRef[T any] struct {
	ref Ref[T]

	result  T
	err     error
	mu      RWMutex
	start   sync.WaitGroup
	started atomic.Bool
	done    atomic.Bool
}

func (r *lazyRef[T]) Get(ctx context.Context) (T, error) {
	var zero T
	if !r.done.Load() {
		if r.started.CompareAndSwap(false, true) {
			ctx, cancel := r.mu.Lock(ctx)
			defer cancel()
			r.start.Done()
			if r.err = ctx.Err(); r.err != nil {
				return zero, r.err
			}

			r.result, r.err = SafeRef(r.ref).Get(ctx)
			r.ref = nil
			r.done.Store(true)
		} else {
			r.start.Wait()
			ctx, cancel := r.mu.RLock(ctx)
			defer cancel()
			if err := ctx.Err(); err != nil {
				return zero, err
			}
		}
	}

	return r.result, r.err
}

func (r *lazyRef[T]) Close() error {
	_, cancel := r.mu.RLock(context.Background())
	defer cancel()
	return Close(r.result)
}

// LazyRef creates a Ref with on-demand execution of the provided Ref.
// The result of the first execution will be cached and returned immediately for all consequent calls.
func LazyRef[T any](ref Ref[T]) Ref[T] {
	r := &lazyRef[T]{ref: ref}
	r.start.Add(1)
	return r
}

func LazyFuncRef[T any](ref FuncRef[T]) Ref[T] {
	return LazyRef[T](ref)
}

type futureRef[T any] struct {
	result T
	err    error
	mu     RWMutex
	done   atomic.Bool
}

func (r *futureRef[T]) Get(ctx context.Context) (T, error) {
	var zero T
	if !r.done.Load() {
		ctx, cancel := r.mu.RLock(ctx)
		defer cancel()
		if err := ctx.Err(); err != nil {
			return zero, err
		}
	}

	return r.result, r.err
}

func (r *futureRef[T]) Close() error {
	_, cancel := r.mu.RLock(context.Background())
	defer cancel()
	return Close(r.result)
}

// FutureRef represents a result of an asynchronous computation that may or may not be available yet.
func FutureRef[T any](ctx context.Context, ref Ref[T]) Ref[T] {
	r := new(futureRef[T])
	ctx, cancel := r.mu.Lock(ctx)
	go func() {
		defer cancel()
		if r.err = ctx.Err(); r.err != nil {
			return
		}

		r.result, r.err = SafeRef(ref).Get(ctx)
		r.done.Store(true)
	}()

	return r
}

func FutureFuncRef[T any](ctx context.Context, ref FuncRef[T]) Ref[T] {
	return FutureRef[T](ctx, ref)
}
