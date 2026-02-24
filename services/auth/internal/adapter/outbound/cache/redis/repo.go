package redis

import (
	"context"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"time"

	"github.com/redis/go-redis/v9"
)

type cacheRepository struct {
	rdb *redis.Client
}

func NewCacheRepository(rdb *redis.Client) outbound.CacheRepository {
	return &cacheRepository{rdb: rdb}
}

func (r *cacheRepository) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return r.rdb.Set(ctx, key, value, ttl).Err()
}

func (r *cacheRepository) Get(ctx context.Context, key string) (string, error) {
	return r.rdb.Get(ctx, key).Result()
}

func (r *cacheRepository) Delete(ctx context.Context, key string) error {
	return r.rdb.Del(ctx, key).Err()
}

func (r *cacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	res, err := r.rdb.Exists(ctx, key).Result()
	return res > 0, err
}

func (r *cacheRepository) IncrWithExpire(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	pipe := r.rdb.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)
	
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return incr.Val(), nil
}