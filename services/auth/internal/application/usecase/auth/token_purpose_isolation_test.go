package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	pkgauth "go-judge-system/pkg/auth"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"
)

func TestTokenPurposeIsolation_RegisterForgotThenVerify(t *testing.T) {
	ctx := context.Background()
	users := &verifyUserRepository{users: map[string]*entity.User{}}
	tokens := newLifecycleTokenRepository()
	generator := &lifecycleTokenGenerator{tokens: []string{"verification-token", "reset-token"}}
	mail := lifecycleMailProvider{}
	registerUC := NewRegisterUseCase(users, mail, generator, tokens, lifecyclePasswordEncoder{})
	forgotUC := NewForgotPasswordUseCase(users, tokens, generator, mail)
	verifyUC := NewVerifyEmailUseCase(generator, tokens, users)
	resetUC := NewResetPasswordUseCase(users, tokens, generator, lifecyclePasswordEncoder{}, lifecycleIATStore{})

	if err := registerUC.Execute(ctx, dto.RegisterRequest{FullName: "Token Test", Username: "token-user", Email: "token-user@example.test", Password: "StrongPass123!"}); err != nil {
		t.Fatalf("register error = %v", err)
	}
	if err := forgotUC.Execute(ctx, dto.ForgotPasswordRequest{Email: "token-user@example.test"}); err != nil {
		t.Fatalf("forgot password error = %v", err)
	}

	if err := resetUC.Execute(ctx, dto.ResetPasswordRequest{Token: "verification-token", NewPassword: "ChangedPass123!", ConfirmPassword: "ChangedPass123!"}); !errors.Is(err, domain.ErrInvalidOrExpiredToken) {
		t.Fatalf("verification token reset error = %v, want ErrInvalidOrExpiredToken", err)
	}
	if err := verifyUC.Execute(ctx, dto.VerifyEmailRequest{Token: "reset-token"}); !errors.Is(err, domain.ErrInvalidOrExpiredToken) {
		t.Fatalf("reset token verify error = %v, want ErrInvalidOrExpiredToken", err)
	}
	if err := verifyUC.Execute(ctx, dto.VerifyEmailRequest{Token: "verification-token"}); err != nil {
		t.Fatalf("verification token must remain valid after reset issuance: %v", err)
	}
	if _, err := tokens.FindByToken(ctx, outbound.TokenPurposeResetPassword, "hashed:reset-token"); err != nil {
		t.Fatalf("reset token must remain valid after verification consumption: %v", err)
	}
}

func TestTokenPurposeIsolation_ForgotResendThenReset(t *testing.T) {
	ctx := context.Background()
	user := newVerifyTestUser(t, "purpose-user", false)
	users := &verifyUserRepository{users: map[string]*entity.User{user.ID: user}}
	tokens := newLifecycleTokenRepository()
	generator := &lifecycleTokenGenerator{tokens: []string{"reset-token", "verification-token"}}
	mail := lifecycleMailProvider{}
	forgotUC := NewForgotPasswordUseCase(users, tokens, generator, mail)
	resendUC := NewResendVerificationUseCase(users, mail, generator, tokens)
	resetUC := NewResetPasswordUseCase(users, tokens, generator, lifecyclePasswordEncoder{}, lifecycleIATStore{})

	if err := forgotUC.Execute(ctx, dto.ForgotPasswordRequest{Email: user.Email}); err != nil {
		t.Fatalf("forgot password error = %v", err)
	}
	if err := resendUC.Execute(ctx, dto.ResendVerificationRequest{Email: user.Email}); err != nil {
		t.Fatalf("resend verification error = %v", err)
	}
	if err := resetUC.Execute(ctx, dto.ResetPasswordRequest{Token: "reset-token", NewPassword: "ChangedPass123!", ConfirmPassword: "ChangedPass123!"}); err != nil {
		t.Fatalf("reset token must remain valid after verification issuance: %v", err)
	}
	if _, err := tokens.FindByToken(ctx, outbound.TokenPurposeVerifyEmail, "hashed:verification-token"); err != nil {
		t.Fatalf("verification token must remain valid after reset consumption: %v", err)
	}
}

func TestLifecycleTokenRepositoryReplacementAndConsumptionArePurposeScoped(t *testing.T) {
	ctx := context.Background()
	repo := newLifecycleTokenRepository()
	const userID = "user-1"

	for _, purpose := range []outbound.TokenPurpose{outbound.TokenPurposeVerifyEmail, outbound.TokenPurposeResetPassword} {
		if err := repo.Save(ctx, purpose, "first-"+string(purpose), userID, time.Minute); err != nil {
			t.Fatal(err)
		}
		if err := repo.Save(ctx, purpose, "second-"+string(purpose), userID, time.Minute); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.FindByToken(ctx, purpose, "first-"+string(purpose)); !errors.Is(err, domain.ErrInvalidOrExpiredToken) {
			t.Fatalf("first %s token error = %v, want invalid", purpose, err)
		}
	}

	if _, err := repo.Consume(ctx, outbound.TokenPurposeVerifyEmail, "second-verify_email"); err != nil {
		t.Fatalf("consume verification token: %v", err)
	}
	if _, err := repo.FindByToken(ctx, outbound.TokenPurposeResetPassword, "second-reset_password"); err != nil {
		t.Fatalf("reset token must survive verification consumption: %v", err)
	}
	if _, err := repo.Consume(ctx, outbound.TokenPurposeResetPassword, "second-reset_password"); err != nil {
		t.Fatalf("consume reset token: %v", err)
	}
	if _, err := repo.Consume(ctx, outbound.TokenPurposeResetPassword, "second-reset_password"); !errors.Is(err, domain.ErrInvalidOrExpiredToken) {
		t.Fatalf("replayed reset token error = %v, want invalid", err)
	}
}

func TestLifecycleTokenRepositoryConcurrentIssuanceIsPurposeScoped(t *testing.T) {
	ctx := context.Background()
	repo := newLifecycleTokenRepository()
	const userID = "concurrent-user"
	var group sync.WaitGroup

	for _, purpose := range []outbound.TokenPurpose{outbound.TokenPurposeVerifyEmail, outbound.TokenPurposeResetPassword} {
		for i := 0; i < 16; i++ {
			group.Add(1)
			go func(p outbound.TokenPurpose, sequence int) {
				defer group.Done()
				if err := repo.Save(ctx, p, fmt.Sprintf("%s-%d", p, sequence), userID, time.Minute); err != nil {
					t.Errorf("Save(%s): %v", p, err)
				}
			}(purpose, i)
		}
	}
	group.Wait()

	for _, purpose := range []outbound.TokenPurpose{outbound.TokenPurposeVerifyEmail, outbound.TokenPurposeResetPassword} {
		repo.mu.Lock()
		latest := repo.latest[purpose][userID]
		repo.mu.Unlock()
		if latest == "" {
			t.Fatalf("no latest token for %s", purpose)
		}
		if _, err := repo.FindByToken(ctx, purpose, latest); err != nil {
			t.Fatalf("latest %s token should be valid: %v", purpose, err)
		}
	}
}

type lifecycleTokenRepository struct {
	mu       sync.Mutex
	tokens   map[outbound.TokenPurpose]map[string]string
	latest   map[outbound.TokenPurpose]map[string]string
	cooldown map[outbound.TokenPurpose]map[string]bool
}

func newLifecycleTokenRepository() *lifecycleTokenRepository {
	return &lifecycleTokenRepository{
		tokens:   map[outbound.TokenPurpose]map[string]string{},
		latest:   map[outbound.TokenPurpose]map[string]string{},
		cooldown: map[outbound.TokenPurpose]map[string]bool{},
	}
}

func (r *lifecycleTokenRepository) Save(_ context.Context, purpose outbound.TokenPurpose, token, identifier string, _ time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.tokens[purpose] == nil {
		r.tokens[purpose] = map[string]string{}
		r.latest[purpose] = map[string]string{}
	}
	r.tokens[purpose][token] = identifier
	r.latest[purpose][identifier] = token
	return nil
}

func (r *lifecycleTokenRepository) FindByToken(_ context.Context, purpose outbound.TokenPurpose, token string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	identifier, ok := r.tokens[purpose][token]
	if !ok || r.latest[purpose][identifier] != token {
		return "", domain.ErrInvalidOrExpiredToken
	}
	return identifier, nil
}

func (r *lifecycleTokenRepository) Consume(ctx context.Context, purpose outbound.TokenPurpose, token string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	identifier, ok := r.tokens[purpose][token]
	if !ok || r.latest[purpose][identifier] != token {
		return "", domain.ErrInvalidOrExpiredToken
	}
	delete(r.tokens[purpose], token)
	delete(r.latest[purpose], identifier)
	return identifier, nil
}

func (r *lifecycleTokenRepository) Delete(ctx context.Context, purpose outbound.TokenPurpose, token string) error {
	_, err := r.Consume(ctx, purpose, token)
	if errors.Is(err, domain.ErrInvalidOrExpiredToken) {
		return nil
	}
	return err
}

func (r *lifecycleTokenRepository) TryAcquireResendCooldown(_ context.Context, purpose outbound.TokenPurpose, identifier string, _ time.Duration) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cooldown[purpose] == nil {
		r.cooldown[purpose] = map[string]bool{}
	}
	if r.cooldown[purpose][identifier] {
		return false, nil
	}
	r.cooldown[purpose][identifier] = true
	return true, nil
}

type lifecycleTokenGenerator struct {
	tokens []string
}

func (g *lifecycleTokenGenerator) Generate(string) string {
	token := g.tokens[0]
	g.tokens = g.tokens[1:]
	return token
}

func (g *lifecycleTokenGenerator) Hash(token string) string { return "hashed:" + token }

type lifecycleMailProvider struct{}

func (lifecycleMailProvider) SendVerificationEmail(context.Context, string, string) error { return nil }
func (lifecycleMailProvider) SendForgotPasswordEmail(context.Context, string, string) error {
	return nil
}

type lifecyclePasswordEncoder struct{}

func (lifecyclePasswordEncoder) HashAndSalt(password []byte) (string, error) {
	return "hashed:" + string(password), nil
}
func (lifecyclePasswordEncoder) ComparePasswords(string, []byte) bool { return false }

type lifecycleIATStore struct{}

func (lifecycleIATStore) SetLogoutAllIAT(context.Context, string, int64) error   { return nil }
func (lifecycleIATStore) GetLogoutAllIAT(context.Context, string) (int64, error) { return 0, nil }

var _ outbound.TokenRepository = (*lifecycleTokenRepository)(nil)
var _ pkgauth.LogoutAllIATStore = lifecycleIATStore{}
