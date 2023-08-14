package etl

import (
	"fmt"
	"log/slog"

	"github.com/jfk9w/hoarder/internal/util"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

type Logger struct {
	*slog.Logger
	desc string
}

func (l *Logger) Error(errs *error, err error, desc string) bool {
	if err == nil {
		return false
	}

	l.Logger.Error(desc, util.Error(err))
	if ldesc := l.desc; ldesc != "" {
		desc = fmt.Sprintf("%s: %s", ldesc, desc)
	}

	return multierr.AppendInto(errs, errors.New(desc))
}

func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		Logger: l.Logger.With(args...),
	}
}

func (l *Logger) WithDesc(key, value string) *Logger {
	return &Logger{
		Logger: l.Logger.With(slog.String(key, value)),
		desc:   fmt.Sprintf("%s %s", key, value),
	}
}
