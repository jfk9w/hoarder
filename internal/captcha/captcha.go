package captcha

import (
	"context"

	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/rucaptcha-api"
	"github.com/pkg/errors"
)

type Config struct {
	RucaptchaKey string `yaml:"rucaptchaKey,omitempty" doc:"API-ключ для сервиса rucaptcha.com."`
}

type TokenProvider interface {
	GetCaptchaToken(ctx context.Context, userAgent, siteKey, pageURL string) (string, error)
}

func NewTokenProvider(cfg Config, clock based.Clock) (TokenProvider, error) {
	if key := cfg.RucaptchaKey; key != "" {
		client, err := rucaptcha.NewClient(rucaptcha.ClientParams{
			Config: rucaptcha.Config{
				Key: key,
			},
			Clock: clock,
		})

		if err != nil {
			return nil, errors.Wrap(err, "create rucaptcha client")
		}

		return &rucaptchaTokenProvider{client: client}, nil
	}

	return nil, nil
}
