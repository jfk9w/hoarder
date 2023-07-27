package lkdr

import (
	"context"

	"github.com/jfk9w-go/lkdr-api"
	"github.com/jfk9w/hoarder/database"
)

type Credential struct {
	Phone string `yaml:"phone" pattern:"7\\d{10}"`
}

type Config struct {
	DB        database.Config         `yaml:"db" doc:"Database connection settings."`
	DeviceID  string                  `yaml:"deviceId" doc:"Device ID to use when making requests to lkdr-api."`
	UserAgent string                  `yaml:"userAgent" doc:"User agent to use when making requests to lkdr-api."`
	Tenants   map[string][]Credential `yaml:"tenants"`
}

type Client interface {
	Receipt(ctx context.Context, in *lkdr.ReceiptIn) (*lkdr.ReceiptOut, error)
	FiscalData(ctx context.Context, in *lkdr.FiscalDataIn) (*lkdr.FiscalDataOut, error)
}
