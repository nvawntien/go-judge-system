package redis

import (
	"context"
	"time"

	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"

	"github.com/redis/go-redis/v9"
)

type tokenRepository struct {
	rdb *redis.Client
}

func NewTokenRepository(rdb *redis.Client) outbound.TokenRepository {
	return &tokenRepository{rdb: rdb}
}

func (r *tokenRepository) Save(ctx context.Context, hashedToken string, identifier string, ttl time.Duration) error {
	key := "token:" + hashedToken
	return r.rdb.Set(ctx, key, identifier, ttl).Err()
}

func (r *tokenRepository) FindByToken(ctx context.Context, hashedToken string) (string, error) {
	key := "token:" + hashedToken

	val, err := r.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", domain.ErrInvalidOrExpiredToken
	}
	if err != nil {
		return "", err
	}

	return val, nil
}

func (r *tokenRepository) Delete(ctx context.Context, hashedToken string) error {
	key := "token:" + hashedToken
	return r.rdb.Del(ctx, key).Err()
}
