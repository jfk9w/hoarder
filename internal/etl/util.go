package etl

import (
	"context"
	"time"

	"gorm.io/gorm/clause"
)

func Upsert(columns ...string) clause.OnConflict {
	onConflict := clause.OnConflict{
		UpdateAll: true,
	}

	for _, column := range columns {
		onConflict.Columns = append(onConflict.Columns, clause.Column{Name: column})
	}

	return onConflict
}

func WithTimeout(ctx context.Context, timeout time.Duration, fn func(ctx context.Context)) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	fn(ctx)
}
