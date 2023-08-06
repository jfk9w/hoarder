package main

import (
	"context"
	"os"
	"syscall"

	"github.com/AlekSi/pointer"
	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/confi"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/jfk9w/hoarder/internal/captcha"
	"github.com/jfk9w/hoarder/internal/etl"
	"github.com/jfk9w/hoarder/internal/etl/lkdr"
	"github.com/jfk9w/hoarder/internal/etl/tinkoff"
	"github.com/jfk9w/hoarder/internal/iface/xmpp"
)

type Config struct {
	Schema string `yaml:"$schema,omitempty" default:"https://raw.githubusercontent.com/jfk9w/hoarder/master/config/schema.yaml"`

	Dump *struct {
		Schema bool `yaml:"schema,omitempty" doc:"Вывод схемы конфигурации в YAML."`
		Values bool `yaml:"values,omitempty" doc:"Вывод значений конфигурации по умолчанию в JSON."`
	} `yaml:"dump,omitempty" doc:"Вывод параметров конфигурации в стандартный поток вывода.\n\nПредназначены для использования как CLI-параметры."`

	Log struct {
		Level       zapcore.Level `yaml:"level,omitempty" default:"info" doc:"Уровень логирования." enum:"debug,info,warn,error,dpanic,panic,fatal"`
		Encoding    string        `yaml:"encoding,omitempty" default:"console" doc:"Формат логирования." enum:"json,console"`
		OutputPaths []string      `yaml:"outputPaths,omitempty" default:"[stderr]" doc:"Пути логирования." examples:"stdout,stderr,/var/log/hoarder.log"`
	} `yaml:"log,omitempty" doc:"Настройки логирования для библиотеки zap"`

	XMPP *xmpp.Config `yaml:"xmpp,omitempty" doc:"Настройки XMPP-интерфейса."`

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

	logLevel := zap.NewAtomicLevelAt(cfg.Log.Level)
	log := zap.Must(zap.Config{
		Level:       logLevel,
		Encoding:    cfg.Log.Encoding,
		OutputPaths: cfg.Log.OutputPaths,
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:  "message",
			LevelKey:    "level",
			TimeKey:     "time",
			EncodeLevel: zapcore.LowercaseLevelEncoder,
			EncodeTime:  zapcore.TimeEncoderOfLayout("2006-01-02T15:04:05.999"),
		},
	}.Build())
	defer func() {
		if err := log.Sync(); err != nil && !errors.Is(err, syscall.ENOTTY) {
			panic(err)
		}
	}()

	clock := based.StandardClock

	captchaSolver, err := captcha.NewTokenProvider(ctx, cfg.Captcha, clock)
	if err != nil {
		panic(err)
	}

	processors := new(etl.Processors)

	if cfg := cfg.LKDR; cfg != nil {
		log := log.With(zap.String("processor", lkdr.Name))
		processor, err := lkdr.Builder{
			Config:        *cfg,
			Clock:         clock,
			Log:           log,
			CaptchaSolver: captchaSolver,
		}.Build(ctx)
		if err != nil {
			log.Fatal("failed to start", zap.Error(err))
		}

		processors.Add(processor)
	}

	if cfg := cfg.Tinkoff; cfg != nil {
		log := log.With(zap.String("processor", tinkoff.Name))
		processor, err := tinkoff.Builder{
			Config: *cfg,
			Clock:  clock,
			Log:    log,
		}.Build(ctx)
		if err != nil {
			log.Fatal("start", zap.Error(err))
		}

		processors.Add(processor)
	}

	if cfg := cfg.XMPP; cfg != nil {
		log := log.With(zap.String("interface", "xmpp"))
		handler, err := xmpp.Builder{
			Config:    *cfg,
			Processor: processors,
			Log:       log,
		}.Run()
		if err != nil {
			log.Error("start", zap.Error(err))
		} else {
			defer handler.Stop()
		}
	}

	if err := based.AwaitSignal(ctx, syscall.SIGINT, syscall.SIGKILL); err != nil {
		log.Fatal("failed to await signal", zap.Error(err))
	}
}

func dump(value any, codec confi.Codec) {
	if err := codec.Marshal(value, os.Stdout); err != nil {
		panic(err)
	}
}
