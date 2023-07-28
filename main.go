package main

import (
	"context"
	"os"

	"github.com/AlekSi/pointer"

	"github.com/jfk9w-go/based"

	"github.com/jfk9w/hoarder/etl/lkdr"

	"github.com/jfk9w-go/confi"
)

type Config struct {
	Schema string `yaml:"$schema,omitempty" default:"https://raw.githubusercontent.com/jfk9w/hoarder/master/config/schema.yaml"`

	Dump *struct {
		Schema bool `yaml:"schema,omitempty" doc:"Вывод схемы конфигурации в YAML."`
		Values bool `yaml:"values,omitempty" doc:"Вывод значений конфигурации по умолчанию в JSON."`
	} `yaml:"dump,omitempty" doc:"Вывод параметров конфигурации в стандартный поток вывода.\n\nПредназначены для использования как CLI-параметры."`

	LKDR *struct {
		lkdr.Config `yaml:"-,inline"`
		Init        *struct {
			Tenant string `yaml:"tenant" doc:"Пользователь, для которого нужно провести инициализацию."`
		} `yaml:"authorize,omitempty" doc:"Инициализация БД и получение начальных данных."`
	} `yaml:"lkdr,omitempty" doc:"Настройка загрузчиков из сервиса ФНС \"Мои чеки онлайн\"."`

	CaptchaSolver *struct {
		RucaptchaKey *string `yaml:"rucaptchaKey,omitempty" doc:"API-ключ для сервиса rucaptcha.com."`
		Token        *string `yaml:"token,omitempty" doc:"Фиксированный капча-токен для выполнения одноразовой операции."`
	} `yaml:"captchaSolver,omitempty" doc:"Настройки для решения капчи."`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, schema, err := confi.Get[Config](ctx, "hoarder")
	if err != nil {
		panic(err)
	}

	if pointer.Get(cfg.Dump).Schema {
		dump(schema, confi.YAML)
		return
	}

	if pointer.Get(cfg.Dump).Values {
		cfg.Dump = nil
		dump(cfg, confi.JSON)
		return
	}

	clock := based.StandardClock

	if cfg := cfg.LKDR; cfg != nil {
		lkdrProcessor := lkdr.NewProcessor(cfg.Config, clock, nil)
		if cfg := cfg.Init; cfg != nil {
			if err := lkdrProcessor.Process(ctx, cfg.Tenant); err != nil {
				panic(err)
			}

			return
		}
	}
}

func dump(value any, codec confi.Codec) {
	if err := codec.Marshal(value, os.Stdout); err != nil {
		panic(err)
	}
}
