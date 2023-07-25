package loader

import "context"

type Interface interface {
	Load(ctx context.Context)
}
