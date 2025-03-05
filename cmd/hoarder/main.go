package main

import (
	"context"
	"os"
	"syscall"

	"github.com/AlekSi/pointer"
	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/confi"
	"github.com/pkg/errors"

	"github.com/jfk9w/hoarder/internal/captcha"
	"github.com/jfk9w/hoarder/internal/firefly"
	"github.com/jfk9w/hoarder/internal/jobs"
	"github.com/jfk9w/hoarder/internal/jobs/lkdr"
	"github.com/jfk9w/hoarder/internal/jobs/tbank"
	"github.com/jfk9w/hoarder/internal/logs"
	"github.com/jfk9w/hoarder/internal/selenium"
	"github.com/jfk9w/hoarder/internal/triggers"
	"github.com/jfk9w/hoarder/internal/triggers/schedule"
	"github.com/jfk9w/hoarder/internal/triggers/stdin"
	"github.com/jfk9w/hoarder/internal/triggers/telegram"
	"github.com/jfk9w/hoarder/internal/triggers/xmpp"
)

type Config struct {
	Schema string `yaml:"$schema,omitempty" default:"https://raw.githubusercontent.com/jfk9w/hoarder/master/config/schema.json"`

	Dump *struct {
		Schema bool `yaml:"schema,omitempty" doc:"Вывод схемы конфигурации в YAML."`
		Values bool `yaml:"values,omitempty" doc:"Вывод значений конфигурации по умолчанию в JSON."`
	} `yaml:"dump,omitempty" doc:"Вывод параметров конфигурации в стандартный поток вывода.\n\nПредназначены для использования как CLI-параметры."`

	Log logs.Config `yaml:"log,omitempty" doc:"Настройки логирования для библиотеки slog."`

	Firefly *struct {
		firefly.Config `yaml:",inline"`
		Enabled        bool `yaml:"enabled,omitempty" doc:"Включить синхронизацию с Firefly III."`
	} `yaml:"firefly,omitempty" doc:"Настройки подключения к Firefly III."`

	Schedule *struct {
		schedule.Config `yaml:",inline"`
		Enabled         bool `yaml:"enabled,omitempty" doc:"Включить фоновую синхронизацию."`
	} `yaml:"schedule,omitempty" doc:"Настройки фоновой синхронизации."`

	Stdin *struct {
		Enabled bool `yaml:"enabled,omitempty" doc:"Включение интерактивной командной строки."`
	} `yaml:"stdin,omitempty" doc:"Настройки управления через интерактивную командную строку."`

	XMPP *struct {
		xmpp.Config `yaml:",inline"`
		Enabled     bool `yaml:"enabled,omitempty" doc:"Включение XMPP-триггера."`
	} `yaml:"xmpp,omitempty" doc:"Настройки XMPP-триггера."`

	Telegram *struct {
		telegram.Config `yaml:",inline"`
		Enabled         bool `yaml:"enabled,omitempty" doc:"Включение Telegram-триггера."`
	} `yaml:"telegram,omitempty" doc:"Настройки Telegram-триггера."`

	LKDR *struct {
		lkdr.Config `yaml:",inline"`
		Enabled     bool `yaml:"enabled,omitempty" doc:"Включает загрузку данных из сервиса ФНС \"Мои чеки онлайн\"."`
	} `yaml:"lkdr,omitempty" doc:"Настройка загрузки данных из сервиса ФНС \"Мои чеки онлайн\"."`

	Tinkoff *struct {
		tbank.Config `yaml:",inline"`
		Enabled      bool `yaml:"enabled,omitempty" doc:"Включает загрузку данных из Т-Банка."`
	} `yaml:"tinkoff,omitempty" doc:"Настройка загрузки данных из Т-Банка"`

	Selenium *struct {
		selenium.Config `yaml:",inline"`
		Enabled         bool `yaml:"enabled,omitempty" doc:"Включает аутентификацию через Selenium."`
	} `yaml:"selenium,omitempty" doc:"Параметры Selenium."`

	Captcha *captcha.Config `yaml:"captcha,omitempty" doc:"Настройки для решения капчи."`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, schema, err := confi.Get[Config](ctx, "hoarder")
	if err != nil {
		panic(err)
	}

	if pointer.Get(cfg.Dump).Schema {
		dump(schema, confi.JSON)
		return
	}

	if pointer.Get(cfg.Dump).Values {
		cfg.Dump = nil
		dump(cfg, confi.JSON)
		return
	}

	log := logs.Get(cfg.Log)
	defer log.Info("shutdown")

	clock := based.StandardClock

	var fireflyClient firefly.Invoker
	if cfg := cfg.Firefly; pointer.Get(cfg).Enabled {
		fireflyClient, err = firefly.NewDefaultClient(firefly.ClientParams{
			Config: cfg.Config,
		})

		if err != nil {
			panic(errors.Wrap(err, "create firefly registry"))
		}
	}

	var captchaSolver captcha.TokenProvider
	if cfg := cfg.Captcha; cfg != nil {
		captchaSolver, err = captcha.NewTokenProvider(cfg, clock)
		if err != nil {
			panic(errors.Wrap(err, "create captcha solver"))
		}
	}

	var seleniumService *selenium.Service
	if cfg := cfg.Selenium; pointer.Get(cfg).Enabled {
		seleniumService, err = selenium.NewService(selenium.ServiceParams{
			Config: cfg.Config,
		})
		if err != nil {
			panic(errors.Wrap(err, "init selenium service"))
		}

		defer seleniumService.Stop()
	}

	jobs := new(jobs.Registry)

	if cfg := cfg.LKDR; pointer.Get(cfg).Enabled {
		job, err := lkdr.NewJob(ctx, lkdr.JobParams{
			Clock:         clock,
			Logger:        log,
			Config:        cfg.Config,
			CaptchaSolver: captchaSolver,
		})

		if err != nil {
			panic(errors.Wrapf(err, "create %s job", lkdr.JobID))
		}

		jobs.Register(job)
	}

	if cfg := cfg.Tinkoff; pointer.Get(cfg).Enabled {
		job, err := tbank.NewJob(ctx, tbank.JobParams{
			Clock:    clock,
			Logger:   log,
			Config:   cfg.Config,
			Firefly:  fireflyClient,
			Selenium: seleniumService,
		})

		if err != nil {
			panic(errors.Wrapf(err, "create %s job", tbank.JobID))
		}

		defer job.Close()
		jobs.Register(job)
	}

	triggers := triggers.NewRegistry(log)

	if cfg := cfg.Schedule; pointer.Get(cfg).Enabled {
		trigger, err := schedule.NewTrigger(schedule.TriggerParams{
			Clock:  clock,
			Config: cfg.Config,
		})

		if err != nil {
			panic(errors.Wrap(err, "create schedule trigger"))
		}

		triggers.Register(trigger)
	}

	if cfg := cfg.Stdin; pointer.Get(cfg).Enabled {
		trigger, err := stdin.NewTrigger(stdin.TriggerParams{
			Clock: clock,
		})

		if err != nil {
			panic(errors.Wrap(err, "create stdin trigger"))
		}

		triggers.Register(trigger)
	}

	if cfg := cfg.XMPP; pointer.Get(cfg).Enabled {
		trigger, err := xmpp.NewTrigger(xmpp.TriggerParams{
			Clock:  clock,
			Config: cfg.Config,
		})

		if err != nil {
			panic(errors.Wrap(err, "create xmpp trigger"))
		}

		triggers.Register(trigger)
	}

	if cfg := cfg.Telegram; pointer.Get(cfg).Enabled {
		trigger, err := telegram.NewTrigger(telegram.TriggerParams{
			Clock:  clock,
			Config: cfg.Config,
			Logger: log,
		})

		if err != nil {
			panic(errors.Wrap(err, "create telegram trigger"))
		}

		triggers.Register(trigger)
	}

	triggers.Run(ctx, jobs)
	defer triggers.Close()

	if err := based.AwaitSignal(ctx, syscall.SIGINT, syscall.SIGTERM); err != nil {
		panic(err)
	}
}

func dump(value any, codec confi.Codec) {
	if err := codec.Marshal(value, os.Stdout); err != nil {
		panic(err)
	}
}
