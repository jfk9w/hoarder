package main

import (
	"context"
	"os"

	"github.com/jfk9w-go/rucaptcha-api"

	"github.com/jfk9w-go/based"

	"github.com/jfk9w/hoarder/etl/lkdr"

	"github.com/jfk9w-go/confi"
)

type Dump struct {
	Schema bool `yaml:"schema,omitempty" doc:"Dump configuration schema in JSON format."`
	Values bool `yaml:"values,omitempty" doc:"Dump configuration values in JSON format."`
}

type Config struct {
	Schema    string      `yaml:"$schema,omitempty" default:"https://raw.githubusercontent.com/jfk9w/hoarder/master/config/schema.json"`
	Dump      Dump        `yaml:"dump,omitempty"`
	LKDR      lkdr.Config `yaml:"lkdr"`
	Rucaptcha *struct {
		Key string `yaml:"key"`
	} `yaml:"rucaptcha,omitempty"`
	Tenant string `yaml:"tenant"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, schema, err := confi.Get[Config](ctx, "hoarder")
	if err != nil {
		panic(err)
	}

	if cfg.Dump.Schema {
		dump(schema)
		return
	}

	if cfg.Dump.Values {
		cfg.Dump = Dump{}
		dump(cfg)
		return
	}

	var (
		clock           = based.StandardClock
		rucaptchaClient lkdr.RucaptchaClient
	)

	if cfg.Rucaptcha != nil {
		rucaptchaClient, err = rucaptcha.ClientBuilder{
			Clock: clock,
			Config: rucaptcha.Config{
				Key: cfg.Rucaptcha.Key,
			},
		}.Build(ctx)
		if err != nil {
			panic(err)
		}
	}

	lkdrProcessor := lkdr.NewProcessor(cfg.LKDR, based.StandardClock, rucaptchaClient)
	if err := lkdrProcessor.Process(ctx, cfg.Tenant); err != nil {
		panic(err)
	}
}

func dump(value any) {
	if err := confi.JSON.Marshal(value, os.Stdout); err != nil {
		panic(err)
	}
}
