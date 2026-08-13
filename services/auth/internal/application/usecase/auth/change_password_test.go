package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	pkgauth "go-judge-system/pkg/auth"
	"go-judge-system/pkg/middleware"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"

	"github.com/gin-gonic/gin"
)

func TestChangePasswordRevokesExistingSessionsBeforePersistingPassword(t *testing.T) {
	user := testSessionUser()
	user.Password = "hashed:current-secret"
	repo := &passwordChangeRepository{user: user}
	store := &passwordChangeIATStore{}
	uc := NewChangePasswordUseCase(repo, passwordChangeEncoder{}, store)

	err := uc.Execute(context.Background(), pkgauth.Claims{UserID: user.ID}, dto.ChangePasswordRequest{
		CurrentPassword: "current-secret",
		NewPassword:     "new-secret",
		ConfirmPassword: "new-secret",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if store.cutoff == 0 || repo.passwordHash != "hashed:new-secret" || !repo.passwordUpdated {
		t.Fatalf("cutoff=%d password=%q updated=%t", store.cutoff, repo.passwordHash, repo.passwordUpdated)
	}

	// Access tokens issued at or before the new cutoff are rejected by the shared middleware.
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.NewAuthMiddleware(store))
	router.GET("/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-User-ID", user.ID)
	req.Header.Set("X-Token-Iat", strconv.FormatInt(store.cutoff, 10))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("old access token status = %d, want %d", response.Code, http.StatusUnauthorized)
	}

	// Refresh tokens issued at or before the cutoff are rejected before a new token is minted.
	jwt := &sessionJWT{refreshIAT: store.cutoff}
	_, err = NewRefreshTokenUseCase(jwt, repo, store).Execute(context.Background(), "old-refresh")
	if !errors.Is(err, domain.ErrInvalidOrExpiredToken) || jwt.accessCalls != 0 {
		t.Fatalf("old refresh error=%v accessCalls=%d", err, jwt.accessCalls)
	}

	// After the cutoff second, only fresh authentication with the new credential can mint a session.
	freshStore := &passwordChangeIATStore{cutoff: store.cutoff - 1}
	_, err = NewLoginUseCase(repo, passwordChangeEncoder{}, &sessionJWT{}, freshStore).Execute(
		context.Background(), dto.LoginRequest{Identifier: user.Username, Password: "current-secret"},
	)
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("old password login error = %v", err)
	}
	freshJWT := &sessionJWT{}
	login, err := NewLoginUseCase(repo, passwordChangeEncoder{}, freshJWT, freshStore).Execute(
		context.Background(), dto.LoginRequest{Identifier: user.Username, Password: "new-secret"},
	)
	if err != nil || login.AccessToken == "" || freshJWT.accessCalls != 1 {
		t.Fatalf("fresh login=%+v error=%v accessCalls=%d", login, err, freshJWT.accessCalls)
	}
}

func TestChangePasswordFailsWithoutPersistingWhenInvalidationFails(t *testing.T) {
	user := testSessionUser()
	user.Password = "hashed:current-secret"
	repo := &passwordChangeRepository{user: user}
	store := &passwordChangeIATStore{setErr: errors.New("redis unavailable")}

	err := NewChangePasswordUseCase(repo, passwordChangeEncoder{}, store).Execute(
		context.Background(),
		pkgauth.Claims{UserID: user.ID},
		dto.ChangePasswordRequest{CurrentPassword: "current-secret", NewPassword: "new-secret", ConfirmPassword: "new-secret"},
	)
	if !errors.Is(err, domain.ErrInternalServer) || repo.passwordUpdated {
		t.Fatalf("error=%v passwordUpdated=%t", err, repo.passwordUpdated)
	}
}

func TestChangePasswordKeepsSessionsRevokedWhenPasswordPersistenceFails(t *testing.T) {
	user := testSessionUser()
	user.Password = "hashed:current-secret"
	repo := &passwordChangeRepository{user: user, passwordErr: errors.New("database unavailable")}
	store := &passwordChangeIATStore{}

	err := NewChangePasswordUseCase(repo, passwordChangeEncoder{}, store).Execute(
		context.Background(),
		pkgauth.Claims{UserID: user.ID},
		dto.ChangePasswordRequest{CurrentPassword: "current-secret", NewPassword: "new-secret", ConfirmPassword: "new-secret"},
	)
	if !errors.Is(err, domain.ErrInternalServer) || store.cutoff == 0 || !repo.passwordUpdated {
		t.Fatalf("error=%v cutoff=%d passwordUpdated=%t", err, store.cutoff, repo.passwordUpdated)
	}
}

type passwordChangeEncoder struct{}

func (passwordChangeEncoder) HashAndSalt(value []byte) (string, error) {
	return "hashed:" + string(value), nil
}
func (passwordChangeEncoder) ComparePasswords(hash string, value []byte) bool {
	return hash == "hashed:"+string(value)
}

type passwordChangeIATStore struct {
	cutoff int64
	setErr error
}

func (s *passwordChangeIATStore) SetLogoutAllIAT(_ context.Context, _ string, issuedAt int64) error {
	if s.setErr != nil {
		return s.setErr
	}
	s.cutoff = issuedAt
	return nil
}
func (s *passwordChangeIATStore) GetLogoutAllIAT(context.Context, string) (int64, error) {
	return s.cutoff, nil
}

type passwordChangeRepository struct {
	user            *entity.User
	passwordHash    string
	passwordUpdated bool
	passwordErr     error
}

func (r *passwordChangeRepository) CreateUser(context.Context, *entity.User) error { return nil }
func (r *passwordChangeRepository) GetUserByEmail(context.Context, string) (*entity.User, error) {
	return r.user, nil
}
func (r *passwordChangeRepository) GetUserByUsername(context.Context, string) (*entity.User, error) {
	return r.user, nil
}
func (r *passwordChangeRepository) GetUserById(context.Context, string) (*entity.User, error) {
	return r.user, nil
}
func (r *passwordChangeRepository) ListUsers(context.Context, outbound.ListUsersFilter) (outbound.ListUsersResult, error) {
	return outbound.ListUsersResult{}, nil
}
func (r *passwordChangeRepository) SearchPublicUsers(context.Context, outbound.SearchPublicUsersFilter) (outbound.SearchPublicUsersResult, error) {
	return outbound.SearchPublicUsersResult{}, nil
}
func (r *passwordChangeRepository) UpdateUser(context.Context, *entity.User) error { return nil }
func (r *passwordChangeRepository) UpdatePassword(_ context.Context, _ string, passwordHash string, _ time.Time) error {
	r.passwordUpdated = true
	if r.passwordErr != nil {
		return r.passwordErr
	}
	r.passwordHash = passwordHash
	if r.user != nil {
		r.user.Password = passwordHash
	}
	return nil
}
func (r *passwordChangeRepository) UpdateProfile(context.Context, string, outbound.ProfileUpdates) error {
	return nil
}
func (r *passwordChangeRepository) UpdateAvatar(context.Context, string, string, string, time.Time) error {
	return nil
}
func (r *passwordChangeRepository) DeleteUser(context.Context, string) error { return nil }

var _ outbound.UserRepository = (*passwordChangeRepository)(nil)
var _ pkgauth.LogoutAllIATStore = (*passwordChangeIATStore)(nil)
