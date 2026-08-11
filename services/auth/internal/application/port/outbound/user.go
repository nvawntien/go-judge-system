package outbound

import (
	"context"
	"time"

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

// ProfileUpdates contains only fields owned by the profile-edit operation.
type ProfileUpdates struct {
	FullName    *string
	Bio         *string
	Country     *string
	School      *string
	Company     *string
	GithubURL   *string
	WebsiteURL  *string
	LinkedinURL *string
	UpdatedAt   time.Time
}

type UserRepository interface {
	CreateUser(ctx context.Context, user *entity.User) error
	GetUserByEmail(ctx context.Context, email string) (*entity.User, error)
	GetUserByUsername(ctx context.Context, username string) (*entity.User, error)
	GetUserById(ctx context.Context, id string) (*entity.User, error)
	ListUsers(ctx context.Context, filter ListUsersFilter) (ListUsersResult, error)
	UpdateUser(ctx context.Context, user *entity.User) error
	UpdatePassword(ctx context.Context, userID string, passwordHash string, updatedAt time.Time) error
	UpdateProfile(ctx context.Context, userID string, updates ProfileUpdates) error
	UpdateAvatar(ctx context.Context, userID string, avatarURL string, avatarObjectKey string, updatedAt time.Time) error
	DeleteUser(ctx context.Context, id string) error
}
