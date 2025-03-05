package tbank

import (
	"context"
	"fmt"

	tbank "github.com/jfk9w-go/tbank-api"

	"github.com/jfk9w/hoarder/internal/jobs"
)

type authorizer struct {
	askFn jobs.AskFunc
}

func (a authorizer) GetConfirmationCode(ctx context.Context, phone string) (string, error) {
	return a.askFn(ctx, fmt.Sprintf(`Код подтверждения для "Тинькофф" • %s: `, phone))
}

func withAuthorizer(ctx context.Context, askFn jobs.AskFunc) context.Context {
	return tbank.WithAuthorizer(ctx, authorizer{askFn: askFn})
}
