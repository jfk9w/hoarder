package lkdr

import (
	"context"
	"fmt"

	"github.com/jfk9w/hoarder/internal/captcha"
	"github.com/jfk9w/hoarder/internal/etl"
)

type authorizer struct {
	captchaSolver  captcha.TokenProvider
	requestInputFn etl.RequestInputFunc
}

func (a *authorizer) GetCaptchaToken(ctx context.Context, userAgent, siteKey, pageURL string) (string, error) {
	return a.captchaSolver.GetCaptchaToken(ctx, userAgent, siteKey, pageURL)
}

func (a *authorizer) GetConfirmationCode(ctx context.Context, phone string) (string, error) {
	return a.requestInputFn(ctx, fmt.Sprintf(`Код подтверждения для сервиса "Мои чеки онлайн" • %s:`, phone))
}
