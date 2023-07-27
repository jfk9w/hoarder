package lkdr

import (
	"context"

	"github.com/jfk9w-go/rucaptcha-api"
)

type RucaptchaClient interface {
	Solve(ctx context.Context, in rucaptcha.SolveIn) (*rucaptcha.SolveOut, error)
}

type rucaptchaTokenProvider struct {
	client RucaptchaClient
}

func (p *rucaptchaTokenProvider) GetCaptchaToken(ctx context.Context, userAgent, siteKey, pageURL string) (string, error) {
	solved, err := p.client.Solve(ctx, &rucaptcha.YandexSmartCaptchaIn{
		UserAgent: userAgent,
		SiteKey:   siteKey,
		PageURL:   pageURL,
	})

	if err != nil {
		return "", err
	}

	return solved.Answer, nil
}
