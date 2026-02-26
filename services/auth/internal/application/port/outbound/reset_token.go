package outbound

import (
	"context"
	"time"
)

type ResetTokenRepository interface {
	Set(ctx context.Context, hashedToken string, email string, ttl time.Duration) error
	Get(ctx context.Context, hashedToken string) (string, error)
	Delete(ctx context.Context, hashedToken string) error
}
