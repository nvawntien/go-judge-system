package outbound

import (
	"context"
	"time"
)

type CacheRepository interface {
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	IncrWithExpire(ctx context.Context, key string, ttl time.Duration) (int64, error)
}
