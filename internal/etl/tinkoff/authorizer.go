package tinkoff

import (
	"context"
	"fmt"

	"github.com/jfk9w/hoarder/internal/etl"
)

type authorizer struct {
	requestInputFn etl.RequestInputFunc
}

func (a *authorizer) GetConfirmationCode(ctx context.Context, phone string) (string, error) {
	return a.requestInputFn(ctx, fmt.Sprintf("Код подтверждения %s (%s)", Name, phone))
}
