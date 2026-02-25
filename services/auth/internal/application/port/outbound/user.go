package outbound

import (
	"context"
	"go-judge-system/services/auth/internal/domain/entity"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *entity.User) error
	GetUserByEmail(ctx context.Context, email string) (*entity.User, error)
	UpdateUser(ctx context.Context, user *entity.User) error
}
