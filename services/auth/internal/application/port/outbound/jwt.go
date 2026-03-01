package outbound

import (
	"context"
)

type JWTProvider interface {
	GenerateAccessToken(ctx context.Context, userID, username, role string) (string, int, error)
	GenerateRefreshToken(ctx context.Context, userID, username, role string) (string, int, error)
	VerifyAccessToken(ctx context.Context, token string) (string, string, string, error)
	VerifyRefreshToken(ctx context.Context, token string) (string, string, string, error)
}
