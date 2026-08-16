package outbound

import (
	"context"
	"time"
)

// TokenPurpose separates otherwise similar one-time-token lifecycles.
type TokenPurpose string

const (
	TokenPurposeVerifyEmail   TokenPurpose = "verify_email"
	TokenPurposeResetPassword TokenPurpose = "reset_password"
)

// TokenRepository stores and retrieves hashed one-time tokens in cache. Every
// operation is purpose-scoped so verification and password-reset state cannot
// replace or consume one another.
type TokenRepository interface {
	Save(ctx context.Context, purpose TokenPurpose, hashedToken string, identifier string, ttl time.Duration) error
	FindByToken(ctx context.Context, purpose TokenPurpose, hashedToken string) (string, error)
	// Consume atomically verifies that a token is the user's latest token and
	// removes it before returning its identifier. It is for single-use flows
	// whose side effects must not be replayed after a partial failure.
	Consume(ctx context.Context, purpose TokenPurpose, hashedToken string) (string, error)
	Delete(ctx context.Context, purpose TokenPurpose, hashedToken string) error
	TryAcquireResendCooldown(ctx context.Context, purpose TokenPurpose, identifier string, ttl time.Duration) (bool, error)
}
