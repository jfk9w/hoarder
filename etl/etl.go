package etl

import "context"

type Processor interface {
	Process(ctx context.Context, user string) error
}
