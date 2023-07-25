package main

import (
	"context"
	"os"

	"github.com/jfk9w-go/confi"

	"github.com/jfk9w/hoarder/jabber"
)

type Dump struct {
	Schema bool `yaml:"schema,omitempty" doc:"Dump configuration schema in JSON format."`
	Values bool `yaml:"values,omitempty" doc:"Dump configuration values in JSON format."`
}

type Config struct {
	Schema string         `yaml:"$schema,omitempty" default:"https://raw.githubusercontent.com/jfk9w/hoarder/master/config.schema.json"`
	Dump   Dump           `yaml:"dump,omitempty"`
	Jabber *jabber.Config `yaml:"jabber,omitempty"`
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

}

func dump(value any) {
	if err := confi.JSON.Marshal(value, os.Stdout); err != nil {
		panic(err)
	}
}
