package outbound

import (
	"context"
	"time"
)

// AuthAbuseLimiter is a distributed, atomic fixed-window limiter. scope is
// hashed by its Redis adapter; callers must pass normalized identifiers.
type AuthAbuseLimiter interface {
	Allow(ctx context.Context, purpose, scope string, limit int, window time.Duration) (allowed bool, retryAfter time.Duration, err error)
	Count(ctx context.Context, purpose, scope string) (count int64, retryAfter time.Duration, err error)
	Reset(ctx context.Context, purpose, scope string) error
}
