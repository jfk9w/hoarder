package etl

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/jfk9w-go/based"
)

type Processor interface {
	Name() string
	Process(ctx context.Context, username string) error
}

type Builder struct {
	processors []Processor
}

func (b *Builder) Add(processor Processor) {
	b.processors = append(b.processors, processor)
}

func (b *Builder) Build() *Processors {
	return &Processors{
		processors: b.processors,
	}
}

type Processors struct {
	processors []Processor
	userMus    map[string]*sync.Mutex
	mu         based.RWMutex
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

	for _, p := range ps.processors {
		for _, err := range multierr.Errors(p.Process(ctx, username)) {
			errs = multierr.Append(errs, errors.Wrap(err, p.Name()))
		}
	}

	return errs
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
