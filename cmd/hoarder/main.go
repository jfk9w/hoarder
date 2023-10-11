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
	"github.com/jfk9w/hoarder/internal/jobs/tinkoff"
	"github.com/jfk9w/hoarder/internal/logs"
	"github.com/jfk9w/hoarder/internal/triggers"
	"github.com/jfk9w/hoarder/internal/triggers/schedule"
	"github.com/jfk9w/hoarder/internal/triggers/stdin"
)

type Config struct {
	Schema string `yaml:"$schema,omitempty" default:"https://raw.githubusercontent.com/jfk9w/hoarder/master/config/schema.yaml"`

	Dump *struct {
		Schema bool `yaml:"schema,omitempty" doc:"Вывод схемы конфигурации в YAML."`
		Values bool `yaml:"values,omitempty" doc:"Вывод значений конфигурации по умолчанию в JSON."`
	} `yaml:"dump,omitempty" doc:"Вывод параметров конфигурации в стандартный поток вывода.\n\nПредназначены для использования как CLI-параметры."`

	Log     logs.Config     `yaml:"log,omitempty" doc:"Настройки логирования для библиотеки slog."`
	Firefly *firefly.Config `yaml:"firefly,omitempty" doc:"Настройки подключения к Firefly III."`

	//XMPP     *xmpp.Config     `yaml:"xmpp,omitempty" doc:"Настройки XMPP-интерфейса."`
	Schedule *schedule.Config `yaml:"schedule,omitempty" doc:"Настройки фоновой синхронизации."`
	Stdin    bool             `yaml:"stdin,omitempty" doc:"Включение интерактивной командной строки."`

	//LKDR    *lkdr.Config    `yaml:"lkdr,omitempty" doc:"Настройка пайплана для сервиса ФНС \"Мои чеки онлайн\"."`
	Tinkoff *tinkoff.Config `yaml:"tinkoff,omitempty" doc:"Настройка пайплайна для онлайн-банка \"Тинькофф\"."`

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

	log := logs.Get(cfg.Log)
	clock := based.StandardClock

	var fireflyClient firefly.Invoker
	if cfg := cfg.Firefly; cfg != nil {
		fireflyClient, err = firefly.NewDefaultClient(firefly.ClientParams{
			Config: cfg,
		})

		if err != nil {
			panic(errors.Wrap(err, "create firefly registry"))
		}
	}

	jobs := new(jobs.Registry)

	if cfg := cfg.Tinkoff; cfg != nil {
		job, err := tinkoff.NewJob(ctx, tinkoff.JobParams{
			Clock:   clock,
			Logger:  log,
			Config:  cfg,
			Firefly: fireflyClient,
		})

		if err != nil {
			panic(errors.Wrap(err, "create tinkoff job"))
		}

		defer job.Close()
		jobs.Register(job)
	}

	triggers := triggers.NewRegistry(log)

	if cfg := cfg.Schedule; cfg != nil {
		trigger, err := schedule.NewTrigger(schedule.TriggerParams{
			Clock:  clock,
			Config: cfg,
		})

		if err != nil {
			panic(errors.Wrap(err, "create schedule trigger"))
		}

		triggers.Register(trigger)
	}

	if cfg.Stdin {
		trigger, err := stdin.NewTrigger(stdin.TriggerParams{})
		if err != nil {
			panic(errors.Wrap(err, "create stdin trigger"))
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
