package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	pkgauth "go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"
)

func TestLoginRejectsSuspendedUser(t *testing.T) {
	user := testSessionUser()
	user.IsSuspended = true
	jwt := &sessionJWT{}
	uc := NewLoginUseCase(&sessionUserRepository{user: user}, sessionPasswordEncoder{}, jwt, &sessionIATStore{})
	_, err := uc.Execute(context.Background(), dto.LoginRequest{Identifier: user.Username, Password: "correct"})
	if !errors.Is(err, domain.ErrUserSuspended) || jwt.accessCalls != 0 {
		t.Fatalf("error=%v accessCalls=%d", err, jwt.accessCalls)
	}
}

func TestRefreshRejectsSuspendedAndInvalidatedUser(t *testing.T) {
	user := testSessionUser()
	user.IsSuspended = true
	jwt := &sessionJWT{refreshIAT: time.Now().Unix() + 1}
	uc := NewRefreshTokenUseCase(jwt, &sessionUserRepository{user: user}, &sessionIATStore{})
	_, err := uc.Execute(context.Background(), "refresh")
	if !errors.Is(err, domain.ErrUserSuspended) || jwt.accessCalls != 0 {
		t.Fatalf("suspended refresh error=%v accessCalls=%d", err, jwt.accessCalls)
	}

	user.IsSuspended = false
	jwt.accessCalls = 0
	_, err = NewRefreshTokenUseCase(jwt, &sessionUserRepository{user: user}, &sessionIATStore{value: jwt.refreshIAT}).Execute(context.Background(), "refresh")
	if !errors.Is(err, domain.ErrInvalidOrExpiredToken) || jwt.accessCalls != 0 {
		t.Fatalf("invalidated refresh error=%v accessCalls=%d", err, jwt.accessCalls)
	}
}

func TestLoginAfterUnsuspensionUsesFreshAuthenticationWithoutClearingCutoff(t *testing.T) {
	user := testSessionUser()
	cutoff := time.Now().Unix() - 1
	store := &sessionIATStore{value: cutoff}
	jwt := &sessionJWT{}
	res, err := NewLoginUseCase(&sessionUserRepository{user: user}, sessionPasswordEncoder{}, jwt, store).Execute(context.Background(), dto.LoginRequest{Identifier: user.Username, Password: "correct"})
	if err != nil || res.AccessToken == "" || jwt.accessCalls != 1 || store.value != cutoff {
		t.Fatalf("response=%+v error=%v accessCalls=%d cutoff=%d", res, err, jwt.accessCalls, store.value)
	}
}

type sessionUserRepository struct {
	user *entity.User
	err  error
}

func (r *sessionUserRepository) CreateUser(context.Context, *entity.User) error { return nil }
func (r *sessionUserRepository) GetUserByEmail(context.Context, string) (*entity.User, error) {
	return r.user, r.err
}
func (r *sessionUserRepository) GetUserByUsername(context.Context, string) (*entity.User, error) {
	return r.user, r.err
}
func (r *sessionUserRepository) GetUserById(context.Context, string) (*entity.User, error) {
	return r.user, r.err
}
func (r *sessionUserRepository) ListUsers(context.Context, outbound.ListUsersFilter) (outbound.ListUsersResult, error) {
	return outbound.ListUsersResult{}, nil
}
func (r *sessionUserRepository) UpdateUser(context.Context, *entity.User) error { return nil }
func (r *sessionUserRepository) DeleteUser(context.Context, string) error       { return nil }

type sessionPasswordEncoder struct{}

func (sessionPasswordEncoder) HashAndSalt([]byte) (string, error) { return "", nil }
func (sessionPasswordEncoder) ComparePasswords(_ string, provided []byte) bool {
	return string(provided) == "correct"
}

type sessionJWT struct {
	refreshIAT  int64
	accessCalls int
}

func (j *sessionJWT) GenerateAccessToken(context.Context, string, string, rbac.Role) (string, int, error) {
	j.accessCalls++
	return "access", 60, nil
}
func (j *sessionJWT) GenerateRefreshToken(context.Context, string, string, rbac.Role) (string, int, error) {
	return "refresh", 60, nil
}
func (j *sessionJWT) VerifyAccessToken(context.Context, string) (string, string, rbac.Role, error) {
	return "", "", "", nil
}
func (j *sessionJWT) VerifyRefreshToken(context.Context, string) (string, string, rbac.Role, int64, error) {
	return "a1cdbf51-5bd4-416d-a74e-35e3cb7b5f83", "session-user", rbac.RoleUser, j.refreshIAT, nil
}

type sessionIATStore struct {
	value int64
	err   error
}

func (s *sessionIATStore) SetLogoutAllIAT(context.Context, string, int64) error { return s.err }
func (s *sessionIATStore) GetLogoutAllIAT(context.Context, string) (int64, error) {
	return s.value, s.err
}

func testSessionUser() *entity.User {
	return &entity.User{
		ID:       "a1cdbf51-5bd4-416d-a74e-35e3cb7b5f83",
		Username: "session-user",
		Email:    "session@example.test",
		Password: "hash",
		Role:     rbac.RoleUser,
		IsActive: true,
	}
}

var _ outbound.UserRepository = (*sessionUserRepository)(nil)
var _ outbound.JWTProvider = (*sessionJWT)(nil)
var _ pkgauth.LogoutAllIATStore = (*sessionIATStore)(nil)
