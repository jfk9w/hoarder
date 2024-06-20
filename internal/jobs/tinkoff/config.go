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
	Selenium     *struct {
		Enabled   bool     `yaml:"enabled,omitempty" doc:"Включает аутентификацию через Selenium."`
		Browser   string   `yaml:"browser,omitempty" enum:"chrome,chromium" doc:"Протестировано только с Chrome/Chromium."`
		Binary    string   `yaml:"binary,omitempty" doc:"Путь к исполняемому файлу браузера."`
		Args      []string `yaml:"args,omitempty" doc:"Аргументы для запуска браузера." default:"[--headless, --no-sandbox]"`
		URLPrefix string   `yaml:"urlPrefix" doc:"Строка подключения к Selenium." default:"http://127.0.0.1:4444/wd/hub"`
	} `yaml:"selenium,omitempty" doc:"Параметры Selenium для аутентификации."`
}
