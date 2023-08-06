package etl

import (
	"context"
	"sync"

	"github.com/jfk9w-go/based"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

type Processor interface {
	Name() string
	Process(ctx context.Context, username string) error
}

type Processors struct {
	processors []Processor
	userMus    map[string]*sync.Mutex
	mu         based.RWMutex
}

func (ps *Processors) Add(processor Processor) {
	ps.processors = append(ps.processors, processor)
}

func (ps *Processors) Process(ctx context.Context, username string) (errs error) {
	mu, err := ps.userMu(ctx, username)
	if err != nil {
		return err
	}

	if !mu.TryLock() {
		return errors.New("already running")
	}

	defer mu.Unlock()

	var (
		errc = make(chan error, len(ps.processors))
		work sync.WaitGroup
	)

	for _, p := range ps.processors {
		work.Add(1)
		go func(p Processor) {
			defer work.Done()
			for _, err := range multierr.Errors(p.Process(ctx, username)) {
				errc <- errors.Wrap(err, p.Name())
			}
		}(p)
	}

	work.Wait()
	close(errc)

	for err := range errc {
		errs = multierr.Append(errs, err)
	}

	return
}

func (ps *Processors) userMu(ctx context.Context, username string) (*sync.Mutex, error) {
	userMu, err := ps.getUserMu(ctx, username)
	if err != nil {
		return nil, err
	}

	if userMu != nil {
		return userMu, nil
	}

	ctx, cancel := ps.mu.Lock(ctx)
	defer cancel()
	if ctx.Err() != nil {
		return nil, err
	}

	if ps.userMus == nil {
		ps.userMus = make(map[string]*sync.Mutex)
	}

	if ps.userMus[username] == nil {
		ps.userMus[username] = new(sync.Mutex)
	}

	return ps.userMus[username], nil
}

func (ps *Processors) getUserMu(ctx context.Context, username string) (*sync.Mutex, error) {
	ctx, cancel := ps.mu.RLock(ctx)
	defer cancel()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if ps.userMus != nil {
		return ps.userMus[username], nil
	}

	return nil, nil
}
