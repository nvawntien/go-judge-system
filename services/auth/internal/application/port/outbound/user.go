package outbound

import (
	"context"

	"go-judge-system/pkg/rbac"
	"go-judge-system/services/auth/internal/domain/entity"
)

type ListUsersFilter struct {
	Search      string
	Role        *rbac.Role
	IsActive    *bool
	IsSuspended *bool
	Limit       int
	Offset      int
}

type ListUsersResult struct {
	Items []*entity.User
	Total int64
}

type UserRepository interface {
	CreateUser(ctx context.Context, user *entity.User) error
	GetUserByEmail(ctx context.Context, email string) (*entity.User, error)
	GetUserByUsername(ctx context.Context, username string) (*entity.User, error)
	GetUserById(ctx context.Context, id string) (*entity.User, error)
	ListUsers(ctx context.Context, filter ListUsersFilter) (ListUsersResult, error)
	UpdateUser(ctx context.Context, user *entity.User) error
	DeleteUser(ctx context.Context, id string) error
}
