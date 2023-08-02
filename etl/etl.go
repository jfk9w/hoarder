package etl

import (
	"context"
)

type Processor interface {
	Process(ctx context.Context, stats *Stats, username string) error
}
