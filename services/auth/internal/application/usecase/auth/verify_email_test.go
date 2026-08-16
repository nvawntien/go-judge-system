package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"
	"go-judge-system/services/auth/internal/domain/valueobject"
)

func TestVerifyEmailActivatesUserAndInvalidatesToken(t *testing.T) {
	user := newVerifyTestUser(t, "user-1", false)
	userRepo := &verifyUserRepository{users: map[string]*entity.User{user.ID: user}}
	tokenRepo := &verifyTokenRepository{tokens: map[string]string{"hashed-valid-token": user.ID}}
	useCase := NewVerifyEmailUseCase(verifyTokenGenerator{}, tokenRepo, userRepo)

	if err := useCase.Execute(context.Background(), dto.VerifyEmailRequest{Token: "valid-token"}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !user.IsActive {
		t.Fatal("expected user to be active")
	}
	if !userRepo.updateCalled {
		t.Fatal("expected user update")
	}
	if !tokenRepo.consumed {
		t.Fatal("expected token consume")
	}
	if err := useCase.Execute(context.Background(), dto.VerifyEmailRequest{Token: "valid-token"}); !errors.Is(err, domain.ErrInvalidOrExpiredToken) {
		t.Fatalf("replayed verification error = %v, want ErrInvalidOrExpiredToken", err)
	}
}

func TestVerifyEmailInvalidOrExpiredToken(t *testing.T) {
	useCase := NewVerifyEmailUseCase(
		verifyTokenGenerator{},
		&verifyTokenRepository{tokens: map[string]string{}},
		&verifyUserRepository{users: map[string]*entity.User{}},
	)

	err := useCase.Execute(context.Background(), dto.VerifyEmailRequest{Token: "missing-token"})
	if !errors.Is(err, domain.ErrInvalidOrExpiredToken) {
		t.Fatalf("Execute() error = %v, want ErrInvalidOrExpiredToken", err)
	}
}

func TestVerifyEmailAlreadyActive(t *testing.T) {
	user := newVerifyTestUser(t, "user-1", true)
	useCase := NewVerifyEmailUseCase(
		verifyTokenGenerator{},
		&verifyTokenRepository{tokens: map[string]string{"hashed-valid-token": user.ID}},
		&verifyUserRepository{users: map[string]*entity.User{user.ID: user}},
	)

	err := useCase.Execute(context.Background(), dto.VerifyEmailRequest{Token: "valid-token"})
	if !errors.Is(err, domain.ErrUserAlreadyActive) {
		t.Fatalf("Execute() error = %v, want ErrUserAlreadyActive", err)
	}
}

func TestVerifyEmailMissingUser(t *testing.T) {
	useCase := NewVerifyEmailUseCase(
		verifyTokenGenerator{},
		&verifyTokenRepository{tokens: map[string]string{"hashed-valid-token": "missing-user"}},
		&verifyUserRepository{users: map[string]*entity.User{}},
	)

	err := useCase.Execute(context.Background(), dto.VerifyEmailRequest{Token: "valid-token"})
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("Execute() error = %v, want ErrUserNotFound", err)
	}
}

func TestVerifyEmailRepositoryError(t *testing.T) {
	tokenRepo := &verifyTokenRepository{findErr: errors.New("redis unavailable")}
	useCase := NewVerifyEmailUseCase(
		verifyTokenGenerator{},
		tokenRepo,
		&verifyUserRepository{users: map[string]*entity.User{}},
	)

	err := useCase.Execute(context.Background(), dto.VerifyEmailRequest{Token: "valid-token"})
	if !errors.Is(err, domain.ErrInternalServer) {
		t.Fatalf("Execute() error = %v, want ErrInternalServer", err)
	}
}

type verifyTokenGenerator struct{}

func (verifyTokenGenerator) Generate(identifier string) string { return "raw-" + identifier }
func (verifyTokenGenerator) Hash(token string) string          { return "hashed-" + token }

type verifyTokenRepository struct {
	tokens   map[string]string
	findErr  error
	consumed bool
}

func (r *verifyTokenRepository) Save(ctx context.Context, _ outbound.TokenPurpose, hashedToken string, identifier string, ttl time.Duration) error {
	r.tokens[hashedToken] = identifier
	return nil
}

func (r *verifyTokenRepository) FindByToken(ctx context.Context, _ outbound.TokenPurpose, hashedToken string) (string, error) {
	if r.findErr != nil {
		return "", r.findErr
	}
	identifier, ok := r.tokens[hashedToken]
	if !ok {
		return "", domain.ErrInvalidOrExpiredToken
	}
	return identifier, nil
}

func (r *verifyTokenRepository) Consume(ctx context.Context, purpose outbound.TokenPurpose, hashedToken string) (string, error) {
	identifier, err := r.FindByToken(ctx, purpose, hashedToken)
	if err != nil {
		return "", err
	}
	if err := r.Delete(ctx, purpose, hashedToken); err != nil {
		return "", err
	}
	r.consumed = true
	return identifier, nil
}

func (r *verifyTokenRepository) Delete(ctx context.Context, _ outbound.TokenPurpose, hashedToken string) error {
	delete(r.tokens, hashedToken)
	return nil
}

func (r *verifyTokenRepository) TryAcquireResendCooldown(ctx context.Context, _ outbound.TokenPurpose, identifier string, ttl time.Duration) (bool, error) {
	return true, nil
}

type verifyUserRepository struct {
	users        map[string]*entity.User
	updateCalled bool
	updateErr    error
}

func (r *verifyUserRepository) CreateUser(ctx context.Context, user *entity.User) error {
	r.users[user.ID] = user
	return nil
}

func (r *verifyUserRepository) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	for _, user := range r.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (r *verifyUserRepository) GetUserByUsername(ctx context.Context, username string) (*entity.User, error) {
	for _, user := range r.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (r *verifyUserRepository) GetUserById(ctx context.Context, id string) (*entity.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}

func (r *verifyUserRepository) ListUsers(context.Context, outbound.ListUsersFilter) (outbound.ListUsersResult, error) {
	return outbound.ListUsersResult{}, nil
}
func (r *verifyUserRepository) SearchPublicUsers(context.Context, outbound.SearchPublicUsersFilter) (outbound.SearchPublicUsersResult, error) {
	return outbound.SearchPublicUsersResult{}, nil
}

func (r *verifyUserRepository) UpdateUser(ctx context.Context, user *entity.User) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.updateCalled = true
	r.users[user.ID] = user
	return nil
}

func (r *verifyUserRepository) UpdatePassword(context.Context, string, string, time.Time) error {
	return nil
}

func (r *verifyUserRepository) UpdateProfile(context.Context, string, outbound.ProfileUpdates) error {
	return nil
}

func (r *verifyUserRepository) UpdateAvatar(context.Context, string, string, string, time.Time) error {
	return nil
}

func (r *verifyUserRepository) DeleteUser(ctx context.Context, id string) error {
	delete(r.users, id)
	return nil
}

func newVerifyTestUser(t *testing.T, id string, active bool) *entity.User {
	t.Helper()

	email, err := valueobject.NewEmail(id + "@example.test")
	if err != nil {
		t.Fatalf("NewEmail() error = %v", err)
	}

	user := entity.NewUser("Test User", id, email, valueobject.NewPasswordFromHash("hashed-password"))
	user.ID = id
	user.IsActive = active
	return user
}
