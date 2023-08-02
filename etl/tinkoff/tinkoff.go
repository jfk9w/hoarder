package tinkoff

import (
	"context"
	"time"

	"github.com/jfk9w-go/tinkoff-api"

	"github.com/jfk9w/hoarder/database"
)

type Credential struct {
	Phone    string `yaml:"phone" pattern:"\\+7\\d{10}" doc:"Номер телефона, на который зарегистрирован аккаунт Тинькофф."`
	Password string `yaml:"password" doc:"Пароль от аккаунта Тинькофф."`
}

type Config struct {
	DB        database.Config         `yaml:"db" doc:"Настройки подключения к БД."`
	BatchSize int                     `yaml:"batchSize,omitempty" doc:"Максимальный размер батчей." default:"100"`
	Overlap   time.Duration           `yaml:"overlap,omitempty" doc:"Продолжительность \"нахлеста\" при обновлении операций." default:"168h"`
	Users     map[string][]Credential `yaml:"users" doc:"Пользователи и их авторизационные данные."`
}

type Client interface {
	AccountsLightIb(ctx context.Context) (tinkoff.AccountsLightIbOut, error)
	Operations(ctx context.Context, in *tinkoff.OperationsIn) (tinkoff.OperationsOut, error)
	ShoppingReceipt(ctx context.Context, in *tinkoff.ShoppingReceiptIn) (*tinkoff.ShoppingReceiptOut, error)
	InvestOperationTypes(ctx context.Context) (*tinkoff.InvestOperationTypesOut, error)
	InvestAccounts(ctx context.Context, in *tinkoff.InvestAccountsIn) (*tinkoff.InvestAccountsOut, error)
	InvestOperations(ctx context.Context, in *tinkoff.InvestOperationsIn) (*tinkoff.InvestOperationsOut, error)
	Close()
}
