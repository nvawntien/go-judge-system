package outbound

import (
	"context"
	"time"
)

// TokenRepository stores and retrieves hashed tokens (verification, password reset) in cache.
type TokenRepository interface {
	Save(ctx context.Context, hashedToken string, identifier string, ttl time.Duration) error
	FindByToken(ctx context.Context, hashedToken string) (string, error)
	// Consume atomically verifies that a token is the user's latest token and
	// removes it before returning its identifier. It is for single-use flows
	// whose side effects must not be replayed after a partial failure.
	Consume(ctx context.Context, hashedToken string) (string, error)
	Delete(ctx context.Context, hashedToken string) error
	TryAcquireResendCooldown(ctx context.Context, identifier string, ttl time.Duration) (bool, error)
}
