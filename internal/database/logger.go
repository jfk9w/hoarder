package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm/logger"

	"github.com/jfk9w/hoarder/internal/logs"
)

type slogLogger struct {
	logger *slog.Logger
	level  logger.LogLevel
}

func slogLevel2logLevel(level slog.Level) logger.LogLevel {
	switch level {
	case slog.LevelDebug:
		return logger.Info
	case slog.LevelInfo:
		return logger.Info
	case slog.LevelWarn:
		return logger.Warn
	case slog.LevelError:
		return logger.Error
	}

	return 0
}

func (l slogLogger) LogMode(level logger.LogLevel) logger.Interface {
	l.level = level
	return l
}

func (l slogLogger) Info(ctx context.Context, msg string, args ...any) {
	l.log(ctx, slog.LevelInfo, fmt.Sprintf(msg, args...))
}

func (l slogLogger) Warn(ctx context.Context, msg string, args ...any) {
	l.log(ctx, slog.LevelWarn, fmt.Sprintf(msg, args...))
}

func (l slogLogger) Error(ctx context.Context, msg string, args ...any) {
	l.log(ctx, slog.LevelError, fmt.Sprintf(msg, args...))
}

func (l slogLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if !l.logger.Enabled(ctx, slog.LevelDebug) {
		return
	}

	sql, rowsAffected := fc()

	level := slog.LevelDebug
	attrs := []slog.Attr{
		slog.Time("begin", begin),
		slog.Int64("rowsAffected", rowsAffected),
	}

	if err != nil {
		level = slog.LevelError
		attrs = append(attrs, logs.Error(err))
	}

	l.log(ctx, level, sql, attrs...)
}

func (l *slogLogger) log(ctx context.Context, level slog.Level, msg string, args ...slog.Attr) {
	if l.level < slogLevel2logLevel(level) {
		return
	}

	l.logger.LogAttrs(ctx, level, msg, args...)
}
