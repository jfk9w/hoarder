package etl

import "context"

type limitedKey struct{}

func WithLimited(ctx context.Context) context.Context {
	return context.WithValue(ctx, limitedKey{}, true)
}

func IsLimited(ctx context.Context) bool {
	flag, _ := ctx.Value(limitedKey{}).(bool)
	return flag
}
