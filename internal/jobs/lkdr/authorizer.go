package lkdr

import (
	"context"
	"fmt"

	"github.com/jfk9w-go/lkdr-api"

	"github.com/jfk9w/hoarder/internal/captcha"
	"github.com/jfk9w/hoarder/internal/jobs"
)

type authorizer struct {
	captcha.TokenProvider
	askFn jobs.AskFunc
}

func (a authorizer) GetConfirmationCode(ctx context.Context, phone string) (string, error) {
	return a.askFn(ctx, fmt.Sprintf(`Код подтверждения для сервиса "Мои чеки онлайн" • %s:`, phone))
}

func withAuthorizer(captchaSolver captcha.TokenProvider) func(ctx context.Context, askFn jobs.AskFunc) context.Context {
	return func(ctx context.Context, askFn jobs.AskFunc) context.Context {
		if captchaSolver == nil {
			return ctx
		}

		return lkdr.WithAuthorizer(ctx, authorizer{
			TokenProvider: captchaSolver,
			askFn:         askFn,
		})
	}
}
