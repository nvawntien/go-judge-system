package redis

import (
	"context"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"time"

	"github.com/redis/go-redis/v9"
)

type resetTokenRepository struct {
	rdb *redis.Client
}

func NewResetTokenRepository(rdb *redis.Client) outbound.ResetTokenRepository {
	return &resetTokenRepository{rdb: rdb}
}

func (r *resetTokenRepository) Save(ctx context.Context, hashedToken string, email string, ttl time.Duration) error {
	key := "reset_token:" + hashedToken
	return r.rdb.Set(ctx, key, email, ttl).Err()
}

func (r *resetTokenRepository) FindEmailByToken(ctx context.Context, hashedToken string) (string, error) {
	key := "reset_token:" + hashedToken

	email, err := r.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", domain.ErrInvalidOrExpiredToken
	}
	if err != nil {
		return "", err
	}

	return email, nil
}

func (r *resetTokenRepository) Delete(ctx context.Context, hashedToken string) error {
	key := "reset_token:" + hashedToken
	return r.rdb.Del(ctx, key).Err()
}
