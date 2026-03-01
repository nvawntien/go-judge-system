package outbound

import (
	"context"
	"time"
)

type ResetTokenRepository interface {
	Save(ctx context.Context, hashedToken string, email string, ttl time.Duration) error
	FindEmailByToken(ctx context.Context, hashedToken string) (string, error)
	Delete(ctx context.Context, hashedToken string) error
}
