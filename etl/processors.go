package etl

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
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

func (b Builder) Build() *Processors {
	return &Processors{
		processors: b.processors,
	}
}

type Processors struct {
	processors []Processor
}

func (p *Processors) Process(ctx context.Context, username string) (errs error) {
	for _, processor := range p.processors {
		if err := processor.Process(ctx, username); err != nil {
			errs = multierr.Append(errs, errors.Wrapf(err, "process %s", processor.Name()))
		}
	}

	return errs
}
