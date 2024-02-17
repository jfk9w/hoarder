package firefly

import (
	"github.com/jfk9w-go/based"
)

//go:generate ogen -clean -no-server -ct-alias application/vnd.api+json=application/json -target . -package firefly ./firefly-iii-2.0.9-v1-fix.yaml

type ClientParams struct {
	Config *Config `validate:"required"`
}

func wrapClient(cfg *clientConfig) {
	cfg.Client = httpClient{cfg.Client}
}

func NewDefaultClient(params ClientParams) (Invoker, error) {
	if err := based.Validate(params); err != nil {
		return nil, err
	}

	wrapClient := optionFunc[clientConfig](wrapClient)
	return NewClient(params.Config.ServerURL, params.Config, wrapClient)
}
