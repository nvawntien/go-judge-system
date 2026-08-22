package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go-judge-system/services/auth/internal/application/port/outbound"
)

type authAbuseLimiter struct{ rdb *redis.Client }

func NewAuthAbuseLimiter(rdb *redis.Client) outbound.AuthAbuseLimiter {
	return &authAbuseLimiter{rdb: rdb}
}

// NewAbuseAdmission exposes atomic, all-or-nothing multi-scope admission.
// It is intentionally separate from NewAuthAbuseLimiter because current Auth
// use cases only need the single-scope limiter contract.
func NewAbuseAdmission(rdb *redis.Client) outbound.AbuseAdmission {
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

// admissionScript first checks every fixed-window key and the optional
// cooldown key. Only after every constraint is available does it mutate any
// key, so a denied admission never partially consumes a quota.
var admissionScript = redis.NewScript(`
local count = tonumber(ARGV[1])
local retry_after = 0

for i = 1, count do
  local limit = tonumber(ARGV[2 + ((i - 1) * 2)])
  local window = tonumber(ARGV[3 + ((i - 1) * 2)])
  local current = tonumber(redis.call('GET', KEYS[i]) or '0')
  if current >= limit then
    local ttl = redis.call('PTTL', KEYS[i])
    if ttl <= 0 then ttl = window end
    if ttl > retry_after then retry_after = ttl end
  end
end

local cooldown_index = count + 1
local cooldown_ttl_arg = 2 + (count * 2)
if #KEYS == cooldown_index and redis.call('EXISTS', KEYS[cooldown_index]) == 1 then
  local ttl = redis.call('PTTL', KEYS[cooldown_index])
  if ttl <= 0 then ttl = tonumber(ARGV[cooldown_ttl_arg]) end
  if ttl > retry_after then retry_after = ttl end
end

if retry_after > 0 then return {0, retry_after} end

for i = 1, count do
  local window = tonumber(ARGV[3 + ((i - 1) * 2)])
  local value = redis.call('INCR', KEYS[i])
  if value == 1 then redis.call('PEXPIRE', KEYS[i], window) end
end

if #KEYS == cooldown_index then
  redis.call('SET', KEYS[cooldown_index], '1', 'PX', ARGV[cooldown_ttl_arg])
end

return {1, 0}`)

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

func (l *authAbuseLimiter) Acquire(ctx context.Context, request outbound.AdmissionRequest) (outbound.AdmissionResult, error) {
	keys, args, err := admissionArguments(request)
	if err != nil {
		return outbound.AdmissionResult{}, err
	}
	values, err := admissionScript.Run(ctx, l.rdb, keys, args...).Int64Slice()
	if err != nil {
		return outbound.AdmissionResult{}, err
	}
	if len(values) != 2 {
		return outbound.AdmissionResult{}, fmt.Errorf("unexpected abuse admission response length: %d", len(values))
	}
	return outbound.AdmissionResult{
		Allowed:    values[0] == 1,
		RetryAfter: time.Duration(values[1]) * time.Millisecond,
	}, nil
}

func admissionArguments(request outbound.AdmissionRequest) ([]string, []interface{}, error) {
	if len(request.Scopes) == 0 && request.Cooldown == nil {
		return nil, nil, fmt.Errorf("abuse admission requires at least one constraint")
	}

	keys := make([]string, 0, len(request.Scopes)+1)
	args := make([]interface{}, 0, 1+(len(request.Scopes)*2)+1)
	args = append(args, len(request.Scopes))
	seen := make(map[string]struct{}, len(request.Scopes)+1)
	appendKey := func(purpose, scope string) error {
		if purpose == "" {
			return fmt.Errorf("abuse admission purpose is required")
		}
		if scope == "" {
			return fmt.Errorf("abuse admission scope is required")
		}
		key := authAbuseKey(purpose, scope)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate abuse admission key for purpose %q", purpose)
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
		return nil
	}

	for _, scope := range request.Scopes {
		if scope.Limit <= 0 {
			return nil, nil, fmt.Errorf("abuse admission limit must be positive")
		}
		if scope.Window <= 0 || scope.Window.Milliseconds() <= 0 {
			return nil, nil, fmt.Errorf("abuse admission window must be positive")
		}
		if err := appendKey(scope.Purpose, scope.Scope); err != nil {
			return nil, nil, err
		}
		args = append(args, scope.Limit, scope.Window.Milliseconds())
	}

	if request.Cooldown != nil {
		if request.Cooldown.Duration <= 0 || request.Cooldown.Duration.Milliseconds() <= 0 {
			return nil, nil, fmt.Errorf("abuse admission cooldown duration must be positive")
		}
		if err := appendKey(request.Cooldown.Purpose, request.Cooldown.Scope); err != nil {
			return nil, nil, err
		}
		args = append(args, request.Cooldown.Duration.Milliseconds())
	}

	return keys, args, nil
}

var _ outbound.AuthAbuseLimiter = (*authAbuseLimiter)(nil)
var _ outbound.AbuseAdmission = (*authAbuseLimiter)(nil)
