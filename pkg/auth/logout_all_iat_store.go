package auth

import (
	"context"
	"go-judge-system/pkg/config"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const logoutAllIATKeyPrefix = "auth:logout_all_iat:"

func LogoutAllIATKey(userID string) string {
	return logoutAllIATKeyPrefix + userID
}

type LogoutAllIATStore interface {
	SetLogoutAllIAT(ctx context.Context, userID string, issuedAt int64) error
	GetLogoutAllIAT(ctx context.Context, userID string) (int64, error)
}

type redisLogoutAllIATStore struct {
	refreshTTL time.Duration
	rdb        *redis.Client
}

func NewRedisLogoutAllIATStore(rdb *redis.Client, cfg config.JWTConfig) LogoutAllIATStore {
	return &redisLogoutAllIATStore{
		refreshTTL: cfg.RefreshTTL,
		rdb:        rdb,
	}
}

func (s *redisLogoutAllIATStore) SetLogoutAllIAT(ctx context.Context, userID string, issuedAt int64) error {
	return s.rdb.Set(ctx, LogoutAllIATKey(userID), strconv.FormatInt(issuedAt, 10), s.refreshTTL).Err()
}

func (s *redisLogoutAllIATStore) GetLogoutAllIAT(ctx context.Context, userID string) (int64, error) {
	rawValue, err := s.rdb.Get(ctx, LogoutAllIATKey(userID)).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	issuedAt, err := strconv.ParseInt(rawValue, 10, 64)
	if err != nil {
		return 0, err
	}

	return issuedAt, nil
}
