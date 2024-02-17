package tinkoff

import (
	"context"

	"github.com/jfk9w-go/based"
)

type pingingClient struct {
	Client
	pinger based.Goroutine
}

func (c pingingClient) Close() error {
	c.pinger.Cancel()
	return c.pinger.Join(context.Background())
}
