package lkdr

import "context"

type initKey struct{}

func Init(ctx context.Context) context.Context {
	return context.WithValue(ctx, initKey{}, true)
}

func isInit(ctx context.Context) bool {
	if init, ok := ctx.Value(initKey{}).(bool); ok {
		return init
	}

	return false
}
