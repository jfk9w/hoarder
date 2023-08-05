package lkdr

import (
	"context"
	"time"

	"github.com/jfk9w-go/lkdr-api"

	"github.com/jfk9w/hoarder/internal/database"
)

type Credential struct {
	Phone string `yaml:"phone" pattern:"7\\d{10}" doc:"Номер телефона пользователя."`
}

type Config struct {
	DB        database.Config         `yaml:"db" doc:"Настройки подключения к БД."`
	UserAgent string                  `yaml:"userAgent,omitempty" doc:"Используется для авторизации и обновления токена доступа.\n\nМожно подсмотреть в браузере при попытке авторизации." default:"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"`
	BatchSize int                     `yaml:"batchSize,omitempty" default:"1000" doc:"Количество чеков в одном запросе и количество фискальных данных за одно обновление."`
	Timeout   time.Duration           `yaml:"timeout,omitempty" default:"5m" doc:"Таймаут для запросов."`
	Users     map[string][]Credential `yaml:"users" doc:"Пользователи и их авторизационные данные."`
}

type Client interface {
	Receipt(ctx context.Context, in *lkdr.ReceiptIn) (*lkdr.ReceiptOut, error)
	FiscalData(ctx context.Context, in *lkdr.FiscalDataIn) (*lkdr.FiscalDataOut, error)
}
