package firefly

import (
	"context"
)

type Config struct {
	ServerURL   string `yaml:"serverUrl" doc:"URL сервера Firefly III."`
	AccessToken string `yaml:"accessToken" doc:"Персональный токен доступа."`
}

func (c Config) FireflyIiiAuth(_ context.Context, _ string) (FireflyIiiAuth, error) {
	return FireflyIiiAuth{Token: c.AccessToken}, nil
}
