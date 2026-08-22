package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/redis/go-redis/v9"
	"go-judge-system/services/auth/internal/application/port/outbound"
)

type authAbuseLimiter struct{ rdb *redis.Client }

func NewAuthAbuseLimiter(rdb *redis.Client) outbound.AuthAbuseLimiter {
	return &authAbuseLimiter{rdb: rdb}
}

func authAbuseKey(purpose, scope string) string {
	s := sha256.Sum256([]byte(scope))
	return "auth:abuse:" + purpose + ":" + hex.EncodeToString(s[:])
}

var allowScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
if n == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[1]) end
local ttl = redis.call('PTTL', KEYS[1])
if n > tonumber(ARGV[2]) then return {0, ttl} end
return {1, ttl}`)

func (l *authAbuseLimiter) Allow(ctx context.Context, purpose, scope string, limit int, window time.Duration) (bool, time.Duration, error) {
	if limit <= 0 || window <= 0 {
		return true, 0, nil
	}
	values, err := allowScript.Run(ctx, l.rdb, []string{authAbuseKey(purpose, scope)}, window.Milliseconds(), limit).Int64Slice()
	if err != nil {
		return false, 0, err
	}
	return values[0] == 1, time.Duration(values[1]) * time.Millisecond, nil
}

func (l *authAbuseLimiter) Count(ctx context.Context, purpose, scope string) (int64, time.Duration, error) {
	key := authAbuseKey(purpose, scope)
	count, err := l.rdb.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	ttl, err := l.rdb.PTTL(ctx, key).Result()
	if err != nil {
		return 0, 0, err
	}
	return count, ttl, nil
}
func (l *authAbuseLimiter) Reset(ctx context.Context, purpose, scope string) error {
	return l.rdb.Del(ctx, authAbuseKey(purpose, scope)).Err()
}
