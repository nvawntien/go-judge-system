package outbound

import (
	"context"
	"go-judge-system/pkg/rbac"
)

type JWTProvider interface {
	GenerateAccessToken(ctx context.Context, userID, username string, role rbac.Role) (string, int, error)
	GenerateRefreshToken(ctx context.Context, userID, username string, role rbac.Role) (string, int, error)
	VerifyAccessToken(ctx context.Context, token string) (string, string, rbac.Role, error)
	VerifyRefreshToken(ctx context.Context, token string) (string, string, rbac.Role, int64, error)
}
