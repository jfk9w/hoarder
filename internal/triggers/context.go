package triggers

import (
	"context"

	"github.com/jfk9w/hoarder/internal/common"
)

type Context interface {
	context.Context

	With(key, value any) Context

	Debug(msg string, attrs ...common.Attr)
	Info(msg string, attrs ...common.Attr)
	Warn(msg string, attrs ...common.Attr)
	Error(msg string, err error, attrs ...common.Attr)
}
