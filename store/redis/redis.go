package redis

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type Store struct {
	store *redis.Client
}

func NewWithDb(tx *redis.Client) *Store {
	return &Store{store: tx}
}

func (r *Store) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return r.store.Set(ctx, key, value, ttl).Err()
}

func (r *Store) Get(ctx context.Context, key string) ([]byte, error) {
	return r.store.Get(ctx, key).Bytes()
}
