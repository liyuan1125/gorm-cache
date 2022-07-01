package cache

import (
	"context"
	"time"
)

type (
	queryCacheCtx    struct{}
	queryCacheKeyCtx struct{}
)

func NewKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, queryCacheKeyCtx{}, key)
}

func NewExpiration(ctx context.Context, ttl time.Duration) context.Context {
	return context.WithValue(ctx, queryCacheCtx{}, ttl)
}

func FromExpiration(ctx context.Context) (time.Duration, bool) {
	value := ctx.Value(queryCacheCtx{})

	if value != nil {
		if t, ok := value.(time.Duration); ok {
			return t, true
		}
	}

	return 0, false
}

func FromKey(ctx context.Context) (string, bool) {
	value := ctx.Value(queryCacheKeyCtx{})

	if value != nil {
		if t, ok := value.(string); ok {
			return t, true
		}

	}

	return "", false
}
