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

	"github.com/gin-gonic/gin"
)

func TestResetPasswordRevokesExistingSessionsAndConsumesToken(t *testing.T) {
	user := testSessionUser()
	user.Password = "hashed:current-secret"
	repo := &passwordChangeRepository{user: user}
	tokens := &resetTokenRepository{tokens: map[string]string{"hashed:valid-token": user.ID}}
	store := &passwordChangeIATStore{}
	useCase := NewResetPasswordUseCase(repo, tokens, resetTokenGenerator{}, passwordChangeEncoder{}, store)

	err := useCase.Execute(context.Background(), dto.ResetPasswordRequest{
		Token:           "valid-token",
		NewPassword:     "new-secret",
		ConfirmPassword: "new-secret",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if store.cutoff == 0 || !repo.passwordUpdated || repo.passwordHash != "hashed:new-secret" || !tokens.consumed {
		t.Fatalf("cutoff=%d passwordUpdated=%t password=%q consumed=%t", store.cutoff, repo.passwordUpdated, repo.passwordHash, tokens.consumed)
	}

	// Access tokens issued at or before the reset cutoff are rejected by the
	// shared middleware before the protected handler receives the request.
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

	// Refresh verifies the same cutoff before minting any new token pair.
	jwt := &sessionJWT{refreshIAT: store.cutoff}
	_, err = NewRefreshTokenUseCase(jwt, repo, store).Execute(context.Background(), "old-refresh")
	if !errors.Is(err, domain.ErrInvalidOrExpiredToken) || jwt.accessCalls != 0 {
		t.Fatalf("old refresh error=%v accessCalls=%d", err, jwt.accessCalls)
	}

	// The old credential no longer authenticates. A genuinely fresh login with
	// the new credential succeeds once it is issued after the cutoff second.
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

	err = useCase.Execute(context.Background(), dto.ResetPasswordRequest{
		Token:           "valid-token",
		NewPassword:     "another-secret",
		ConfirmPassword: "another-secret",
	})
	if !errors.Is(err, domain.ErrInvalidOrExpiredToken) {
		t.Fatalf("replayed reset error = %v, want ErrInvalidOrExpiredToken", err)
	}
}

func TestResetPasswordFailsWithoutPersistingWhenSessionInvalidationFails(t *testing.T) {
	user := testSessionUser()
	user.Password = "hashed:current-secret"
	repo := &passwordChangeRepository{user: user}
	tokens := &resetTokenRepository{tokens: map[string]string{"hashed:valid-token": user.ID}}
	store := &passwordChangeIATStore{setErr: errors.New("redis unavailable")}

	err := NewResetPasswordUseCase(repo, tokens, resetTokenGenerator{}, passwordChangeEncoder{}, store).Execute(
		context.Background(), dto.ResetPasswordRequest{Token: "valid-token", NewPassword: "new-secret", ConfirmPassword: "new-secret"},
	)
	if !errors.Is(err, domain.ErrInternalServer) || repo.passwordUpdated || !tokens.consumed {
		t.Fatalf("error=%v passwordUpdated=%t consumed=%t", err, repo.passwordUpdated, tokens.consumed)
	}
}

func TestResetPasswordReturnsInternalWhenTokenStoreIsUnavailable(t *testing.T) {
	user := testSessionUser()
	user.Password = "hashed:current-secret"
	repo := &passwordChangeRepository{user: user}
	tokens := &resetTokenRepository{consumeErr: errors.New("redis unavailable")}
	store := &passwordChangeIATStore{}

	err := NewResetPasswordUseCase(repo, tokens, resetTokenGenerator{}, passwordChangeEncoder{}, store).Execute(
		context.Background(), dto.ResetPasswordRequest{Token: "valid-token", NewPassword: "new-secret", ConfirmPassword: "new-secret"},
	)
	if !errors.Is(err, domain.ErrInternalServer) || repo.passwordUpdated || store.cutoff != 0 || tokens.consumed {
		t.Fatalf("error=%v passwordUpdated=%t cutoff=%d consumed=%t", err, repo.passwordUpdated, store.cutoff, tokens.consumed)
	}
}

func TestResetPasswordKeepsSessionsRevokedWhenPasswordPersistenceFails(t *testing.T) {
	user := testSessionUser()
	user.Password = "hashed:current-secret"
	repo := &passwordChangeRepository{user: user, passwordErr: errors.New("database unavailable")}
	tokens := &resetTokenRepository{tokens: map[string]string{"hashed:valid-token": user.ID}}
	store := &passwordChangeIATStore{}

	err := NewResetPasswordUseCase(repo, tokens, resetTokenGenerator{}, passwordChangeEncoder{}, store).Execute(
		context.Background(), dto.ResetPasswordRequest{Token: "valid-token", NewPassword: "new-secret", ConfirmPassword: "new-secret"},
	)
	if !errors.Is(err, domain.ErrInternalServer) || store.cutoff == 0 || !repo.passwordUpdated || !tokens.consumed {
		t.Fatalf("error=%v cutoff=%d passwordUpdated=%t consumed=%t", err, store.cutoff, repo.passwordUpdated, tokens.consumed)
	}
}

func TestResetPasswordPreservesAccountActivationAndSuspensionState(t *testing.T) {
	tests := []struct {
		name       string
		active     bool
		suspended  bool
		loginError error
	}{
		{name: "unverified", active: false, loginError: domain.ErrUserInactive},
		{name: "suspended", active: true, suspended: true, loginError: domain.ErrUserSuspended},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			user := testSessionUser()
			user.Password = "hashed:current-secret"
			user.IsActive = test.active
			user.IsSuspended = test.suspended
			repo := &passwordChangeRepository{user: user}
			tokens := &resetTokenRepository{tokens: map[string]string{"hashed:valid-token": user.ID}}
			store := &passwordChangeIATStore{}

			err := NewResetPasswordUseCase(repo, tokens, resetTokenGenerator{}, passwordChangeEncoder{}, store).Execute(
				context.Background(), dto.ResetPasswordRequest{Token: "valid-token", NewPassword: "new-secret", ConfirmPassword: "new-secret"},
			)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if user.IsActive != test.active || user.IsSuspended != test.suspended {
				t.Fatalf("active=%t suspended=%t", user.IsActive, user.IsSuspended)
			}

			_, err = NewLoginUseCase(repo, passwordChangeEncoder{}, &sessionJWT{}, &passwordChangeIATStore{cutoff: store.cutoff - 1}).Execute(
				context.Background(), dto.LoginRequest{Identifier: user.Username, Password: "new-secret"},
			)
			if !errors.Is(err, test.loginError) {
				t.Fatalf("fresh login error = %v, want %v", err, test.loginError)
			}
		})
	}
}

type resetTokenGenerator struct{}

func (resetTokenGenerator) Generate(identifier string) string { return "raw-" + identifier }
func (resetTokenGenerator) Hash(token string) string          { return "hashed:" + token }

type resetTokenRepository struct {
	tokens     map[string]string
	consumeErr error
	consumed   bool
}

func (r *resetTokenRepository) Save(context.Context, outbound.TokenPurpose, string, string, time.Duration) error {
	return nil
}
func (r *resetTokenRepository) FindByToken(_ context.Context, _ outbound.TokenPurpose, hashedToken string) (string, error) {
	identifier, ok := r.tokens[hashedToken]
	if !ok {
		return "", domain.ErrInvalidOrExpiredToken
	}
	return identifier, nil
}
func (r *resetTokenRepository) Consume(ctx context.Context, purpose outbound.TokenPurpose, hashedToken string) (string, error) {
	if r.consumeErr != nil {
		return "", r.consumeErr
	}
	identifier, err := r.FindByToken(ctx, purpose, hashedToken)
	if err != nil {
		return "", err
	}
	delete(r.tokens, hashedToken)
	r.consumed = true
	return identifier, nil
}
func (r *resetTokenRepository) Delete(context.Context, outbound.TokenPurpose, string) error {
	return nil
}
func (r *resetTokenRepository) TryAcquireResendCooldown(context.Context, outbound.TokenPurpose, string, time.Duration) (bool, error) {
	return true, nil
}

var _ pkgauth.LogoutAllIATStore = (*passwordChangeIATStore)(nil)
