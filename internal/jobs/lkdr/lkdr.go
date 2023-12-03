package lkdr

import (
	"context"

	"github.com/jfk9w-go/lkdr-api"
)

type Client interface {
	Receipt(ctx context.Context, in *lkdr.ReceiptIn) (*lkdr.ReceiptOut, error)
	FiscalData(ctx context.Context, in *lkdr.FiscalDataIn) (*lkdr.FiscalDataOut, error)
}

type ClientFactory func(params lkdr.ClientParams) (Client, error)

var defaultClientFactory ClientFactory = func(params lkdr.ClientParams) (Client, error) {
	return lkdr.NewClient(params)
}
