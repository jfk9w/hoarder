package lkdr

import (
	"context"

	"github.com/jfk9w-go/lkdr-api"

	"github.com/jfk9w/hoarder/database"
)

type Credential struct {
	Phone string `yaml:"phone" pattern:"7\\d{10}" doc:"Номер телефона пользователя."`
}

type Config struct {
	DB        database.Config         `yaml:"db" doc:"Настройки подключения к БД."`
	DeviceID  string                  `yaml:"deviceId" doc:"Используется для авторизации и обновления токена доступа.\n\nМожно подсмотреть в браузере при попытке авторизации."`
	UserAgent string                  `yaml:"userAgent" doc:"Используется для авторизации и обновления токена доступа.\n\nМожно подсмотреть в браузере при попытке авторизации."`
	BatchSize int                     `yaml:"batchSize,omitempty" default:"1000" doc:"Количество чеков в одном запросе и количество фискальных данных за одно обновление."`
	Tenants   map[string][]Credential `yaml:"tenants" doc:"Пользователи и их авторизационные данные."`
}

type Client interface {
	Receipt(ctx context.Context, in *lkdr.ReceiptIn) (*lkdr.ReceiptOut, error)
	FiscalData(ctx context.Context, in *lkdr.FiscalDataIn) (*lkdr.FiscalDataOut, error)
}
