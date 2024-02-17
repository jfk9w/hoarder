package tinkoff

import (
	"time"

	"github.com/jfk9w/hoarder/internal/database"
)

type Credential struct {
	Phone    string `yaml:"phone" pattern:"7\\d{10}" doc:"Номер телефона, на который зарегистрирован аккаунт Тинькофф."`
	Password string `yaml:"password" doc:"Пароль от аккаунта Тинькофф."`
}

type Config struct {
	Database     database.Config         `yaml:"database" doc:"Настройки подключения к БД."`
	BatchSize    int                     `yaml:"batchSize,omitempty" doc:"Максимальный размер батчей." default:"100"`
	Overlap      time.Duration           `yaml:"overlap,omitempty" doc:"Продолжительность \"нахлеста\" при обновлении операций." default:"168h"`
	WithReceipts bool                    `yaml:"withReceipts,omitempty" doc:"Включить синхронизацию чеков." default:"true"`
	Users        map[string][]Credential `yaml:"users" doc:"Пользователи и их авторизационные данные."`
}
