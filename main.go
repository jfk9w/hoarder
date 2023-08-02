package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jfk9w/hoarder/etl"

	"github.com/jfk9w/hoarder/etl/tinkoff"

	"github.com/AlekSi/pointer"

	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/confi"

	"github.com/jfk9w/hoarder/captcha"
	"github.com/jfk9w/hoarder/etl/lkdr"
)

type Config struct {
	Schema string `yaml:"$schema,omitempty" default:"https://raw.githubusercontent.com/jfk9w/hoarder/master/config/schema.yaml"`

	Dump *struct {
		Schema bool `yaml:"schema,omitempty" doc:"Вывод схемы конфигурации в YAML."`
		Values bool `yaml:"values,omitempty" doc:"Вывод значений конфигурации по умолчанию в JSON."`
	} `yaml:"dump,omitempty" doc:"Вывод параметров конфигурации в стандартный поток вывода.\n\nПредназначены для использования как CLI-параметры."`

	LKDR *struct {
		lkdr.Config `yaml:"-,inline"`
		Init        string `yaml:"init,omitempty" doc:"Пользователь, для которого нужно провести инициализацию."`
	} `yaml:"lkdr,omitempty" doc:"Настройка загрузчиков из сервиса ФНС \"Мои чеки онлайн\"."`

	Tinkoff *struct {
		tinkoff.Config `yaml:"-,inline"`
		Init           string `yaml:"init,omitempty" doc:"Пользователь, для которого нужно провести инициализацию."`
	} `yaml:"tinkoff,omitempty" doc:"Настройка загрузчиков из онлайн-банка \"Тинькофф\"."`

	Captcha captcha.Config `yaml:"captcha,omitempty" doc:"Настройки для решения капчи."`
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
	captchaTokenProvider, err := captcha.NewTokenProvider(ctx, clock, cfg.Captcha)
	if err != nil {
		panic(err)
	}

	if cfg := cfg.LKDR; cfg != nil {
		processor := lkdr.NewProcessor(cfg.Config, clock, captchaTokenProvider)
		if user := cfg.Init; user != "" {
			ctx := lkdr.Init(ctx)
			stats := new(etl.Stats)
			if err := processor.Process(ctx, stats, user); err != nil {
				panic(err)
			}

			fmt.Println(stats)
			return
		}
	}

	if cfg := cfg.Tinkoff; cfg != nil {
		processor := tinkoff.NewProcessor(cfg.Config, clock)
		if user := cfg.Init; user != "" {
			stats := new(etl.Stats)
			if err := processor.Process(ctx, stats, user); err != nil {
				panic(err)
			}

			fmt.Println(stats)
			return
		}
	}
}

func dump(value any, codec confi.Codec) {
	if err := codec.Marshal(value, os.Stdout); err != nil {
		panic(err)
	}
}
