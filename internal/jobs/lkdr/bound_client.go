package lkdr

import (
	"context"
	"time"

	"github.com/jfk9w-go/lkdr-api"
)

type boundClient struct {
	client  Client
	timeout time.Duration
}

func (c *boundClient) Receipt(ctx context.Context, in *lkdr.ReceiptIn) (*lkdr.ReceiptOut, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.Receipt(ctx, in)
}

func (c *boundClient) FiscalData(ctx context.Context, in *lkdr.FiscalDataIn) (*lkdr.FiscalDataOut, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.FiscalData(ctx, in)
}
