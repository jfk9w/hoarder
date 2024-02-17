package firefly

import (
	"go.uber.org/multierr"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/firefly"
	"github.com/jfk9w/hoarder/internal/jobs"
)

type All struct {
	Phones    []string
	BatchSize int
}

func (s All) TableName() string {
	return "base"
}

func (s All) Sync(ctx jobs.Context, db database.DB, client firefly.Invoker) (ls []Interface, errs error) {
	for _, sync := range []Interface{
		Categories{},
		Currencies{},
	} {
		ctx := ctx.With("subentity", sync.TableName())
		_, err := sync.Sync(ctx, db, client)
		_ = multierr.AppendInto(&errs, err)
	}

	if errs != nil {
		return
	}

	for _, phone := range s.Phones {
		ls = append(ls, accounts{
			phone:     phone,
			batchSize: s.BatchSize,
		})
	}

	return
}
