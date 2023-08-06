package database

import (
	"context"
	"time"

	"gorm.io/gorm/logger"
)

type noopLogger struct{}

func (noopLogger) LogMode(logger.LogLevel) logger.Interface {
	return noopLogger{}
}

func (noopLogger) Info(context.Context, string, ...interface{}) {}

func (noopLogger) Warn(context.Context, string, ...interface{}) {}

func (noopLogger) Error(context.Context, string, ...interface{}) {}

func (noopLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {

}
