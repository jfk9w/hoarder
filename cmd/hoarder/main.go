package main

import (
	"context"
	"log/slog"
	"os"
	"syscall"

	"github.com/AlekSi/pointer"
	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/confi"

	"github.com/jfk9w/hoarder/internal/captcha"
	"github.com/jfk9w/hoarder/internal/etl"
	"github.com/jfk9w/hoarder/internal/etl/lkdr"
	"github.com/jfk9w/hoarder/internal/etl/tinkoff"
	"github.com/jfk9w/hoarder/internal/iface/schedule"
	"github.com/jfk9w/hoarder/internal/iface/stdin"
	"github.com/jfk9w/hoarder/internal/iface/xmpp"
	"github.com/jfk9w/hoarder/internal/log"
)

type Config struct {
	Schema string `yaml:"$schema,omitempty" default:"https://raw.githubusercontent.com/jfk9w/hoarder/master/config/schema.yaml"`

	Dump *struct {
		Schema bool `yaml:"schema,omitempty" doc:"Вывод схемы конфигурации в YAML."`
		Values bool `yaml:"values,omitempty" doc:"Вывод значений конфигурации по умолчанию в JSON."`
	} `yaml:"dump,omitempty" doc:"Вывод параметров конфигурации в стандартный поток вывода.\n\nПредназначены для использования как CLI-параметры."`

	Log log.Config `yaml:"log,omitempty" doc:"Настройки логирования для библиотеки slog."`

	XMPP     *xmpp.Config     `yaml:"xmpp,omitempty" doc:"Настройки XMPP-интерфейса."`
	Schedule *schedule.Config `yaml:"schedule,omitempty" doc:"Настройки фоновой синхронизации."`
	Stdin    bool             `yaml:"stdin,omitempty" doc:"Включение интерактивной командной строки."`

	LKDR    *lkdr.Config    `yaml:"lkdr,omitempty" doc:"Настройка загрузчиков из сервиса ФНС \"Мои чеки онлайн\"."`
	Tinkoff *tinkoff.Config `yaml:"tinkoff,omitempty" doc:"Настройка загрузчиков из онлайн-банка \"Тинькофф\"."`

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

	log := log.Get(cfg.Log)
	clock := based.StandardClock

	captchaSolver, err := captcha.NewTokenProvider(ctx, cfg.Captcha, clock)
	if err != nil {
		panic(err)
	}

	registry := new(etl.Registry)

	if cfg := cfg.LKDR; cfg != nil {
		builder := lkdr.Builder{
			Config:        *cfg,
			Clock:         clock,
			CaptchaSolver: captchaSolver,
		}

		pipeline := must(builder.Build(ctx))
		registry.Register("lkdr", pipeline)
	}

	if cfg := cfg.Tinkoff; cfg != nil {
		builder := tinkoff.Builder{
			Config: *cfg,
			Clock:  clock,
		}

		pipeline := must(builder.Build(ctx))
		registry.Register("tinkoff", pipeline)
	}

	if cfg := cfg.XMPP; cfg != nil {
		builder := xmpp.Builder{
			Config:    *cfg,
			Processor: registry,
			Log:       log.With(slog.String("interface", "xmpp")),
		}

		defer must(builder.Run(ctx)).Stop()
	}

	if cfg := cfg.Schedule; cfg != nil {
		builder := schedule.Builder{
			Config:    *cfg,
			Pipelines: registry,
			Log:       log.With(slog.String("interface", "schedule")),
		}

		defer must(builder.Run(ctx))()
	}

	if cfg.Stdin {
		builder := stdin.Builder{
			Pipelines: registry,
			Log:       log.With(slog.String("interface", "stdin")),
		}

		defer must(builder.Run(ctx))
	}

	if err := based.AwaitSignal(ctx, syscall.SIGINT, syscall.SIGTERM); err != nil {
		panic(err)
	}
}

func must[R any](result R, err error) R {
	if err != nil {
		panic(err)
	}

	return result
}

func dump(value any, codec confi.Codec) {
	if err := codec.Marshal(value, os.Stdout); err != nil {
		panic(err)
	}
}
