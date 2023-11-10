package triggers

import (
	"context"
	"log/slog"
	"time"

	"github.com/jfk9w/hoarder/internal/jobs"
)

type Context struct {
	std context.Context
	log *slog.Logger
}

func NewContext(ctx context.Context, log *slog.Logger) Context {
	return Context{
		std: ctx,
		log: log,
	}
}

func (ctx Context) Deadline() (time.Time, bool) { return ctx.std.Deadline() }
func (ctx Context) Done() <-chan struct{}       { return ctx.std.Done() }
func (ctx Context) Err() error                  { return ctx.std.Err() }
func (ctx Context) Value(key any) any           { return ctx.std.Value(key) }

func (ctx Context) Debug(msg string, args ...any) { ctx.log.Debug(msg, args...) }
func (ctx Context) Info(msg string, args ...any)  { ctx.log.Info(msg, args...) }
func (ctx Context) Warn(msg string, args ...any)  { ctx.log.Warn(msg, args...) }
func (ctx Context) Error(msg string, args ...any) { ctx.log.Error(msg, args...) }

func (ctx Context) With(key string, value any) Context {
	ctx.log = ctx.log.With(slog.Any(key, value))
	return ctx
}

func (ctx Context) As(userID string) Context {
	return ctx.With("user", userID)
}

func (ctx Context) Job() jobs.Context {
	return jobs.NewContext(ctx.std, ctx.log)
}
