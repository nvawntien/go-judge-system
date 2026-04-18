package outbound

import (
	"context"
	"time"
)

// TokenRepository stores and retrieves hashed tokens (verification, password reset) in cache.
type TokenRepository interface {
	Save(ctx context.Context, hashedToken string, identifier string, ttl time.Duration) error
	FindByToken(ctx context.Context, hashedToken string) (string, error)
	Delete(ctx context.Context, hashedToken string) error
}
