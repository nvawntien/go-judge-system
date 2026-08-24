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

// AdmissionScope describes one fixed-window constraint in an abuse admission.
// Purpose and Scope are adapter-private inputs: the Redis adapter namespaces the
// purpose and hashes the scope before constructing a key.
type AdmissionScope struct {
	Purpose string
	Scope   string
	Limit   int
	Window  time.Duration
}

// CooldownScope describes an optional single-use cooldown constraint that is
// acquired with the fixed-window constraints.
type CooldownScope struct {
	Purpose  string
	Scope    string
	Duration time.Duration
}

// AdmissionRequest contains every constraint that must be acquired together.
type AdmissionRequest struct {
	Scopes   []AdmissionScope
	Cooldown *CooldownScope
}

// AdmissionResult reports whether all requested constraints were acquired.
// RetryAfter is non-zero for a denied request when Redis can calculate it.
type AdmissionResult struct {
	Allowed    bool
	RetryAfter time.Duration
}

// AbuseAdmission atomically acquires every constraint in one request. Either
// all constraints are acquired, or none are mutated. It deliberately remains
// separate from AuthAbuseLimiter so existing single-scope callers do not take
// on a transactional admission dependency.
type AbuseAdmission interface {
	Acquire(ctx context.Context, request AdmissionRequest) (AdmissionResult, error)
}
