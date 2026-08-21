package auth

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-judge-system/pkg/config"
	"go-judge-system/pkg/requestctx"
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"
)

func TestLoginLimiterNormalizesIdentifierAndClearsAfterSuccess(t *testing.T) {
	limiter := newMemoryLimiter()
	policy := config.AuthAbuseConfig{LoginIPLimit: 50, LoginIdentifierLimit: 2, LoginWindow: time.Minute}
	user := testSessionUser()
	uc := NewLoginUseCaseWithAbuse(&sessionUserRepository{user: user}, sessionPasswordEncoder{}, &sessionJWT{}, &sessionIATStore{}, limiter, policy)
	ctx := requestctx.WithClientIP(context.Background(), "203.0.113.1")
	for i := 0; i < 2; i++ {
		_, err := uc.Execute(ctx, dto.LoginRequest{Identifier: " SESSION-USER ", Password: "bad"})
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("attempt %d error=%v", i, err)
		}
	}
	_, err := uc.Execute(ctx, dto.LoginRequest{Identifier: "session-user", Password: "bad"})
	if !errors.Is(err, domain.ErrRateLimitExceeded) {
		t.Fatalf("blocked error=%v", err)
	}
	// A new identifier bucket is independent; a successful login clears its
	// failed-attempt state so the next failure is not throttled.
	_, err = uc.Execute(ctx, dto.LoginRequest{Identifier: "other", Password: "correct"})
	if err != nil {
		t.Fatalf("successful login=%v", err)
	}
}

func TestMemoryLimiterConcurrentAttemptsCannotExceedLimit(t *testing.T) {
	limiter := newMemoryLimiter()
	var allowed atomic.Int32
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, _, err := limiter.Allow(context.Background(), "test", "scope", 5, time.Minute)
			if err != nil {
				t.Error(err)
				return
			}
			if ok {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()
	if allowed.Load() != 5 {
		t.Fatalf("allowed=%d want 5", allowed.Load())
	}
}

func TestResendAndForgotQuotaBlockMailWithoutChangingGenericResult(t *testing.T) {
	user := newVerifyTestUser(t, "quota-user", false)
	users := &verifyUserRepository{users: map[string]*entity.User{user.ID: user}}
	mail := &countingMail{}
	policy := config.AuthAbuseConfig{MailIPHourlyLimit: 10, MailIPDailyLimit: 10, ResendAccountHourlyLimit: 1, ResendAccountDailyLimit: 10, ForgotAccountHourlyLimit: 1, ForgotAccountDailyLimit: 10, EmailCooldown: time.Hour}
	resend := NewResendVerificationUseCaseWithAbuse(users, mail, &lifecycleTokenGenerator{tokens: []string{"verify-1"}}, newLifecycleTokenRepository(), newMemoryLimiter(), policy)
	resendCtx := requestctx.WithClientIP(context.Background(), "203.0.113.2")
	for i := 0; i < 2; i++ {
		if err := resend.Execute(resendCtx, dto.ResendVerificationRequest{Email: user.Email}); err != nil {
			t.Fatalf("resend %d error=%v", i, err)
		}
	}
	if mail.verify != 1 {
		t.Fatalf("verification emails=%d want 1", mail.verify)
	}
	if err := resend.Execute(requestctx.WithClientIP(context.Background(), "203.0.113.3"), dto.ResendVerificationRequest{Email: "missing@example.test"}); err != nil || mail.verify != 1 {
		t.Fatalf("unknown resend error=%v emails=%d", err, mail.verify)
	}

	forgotMail := &countingMail{}
	forgot := NewForgotPasswordUseCaseWithAbuse(users, newLifecycleTokenRepository(), &lifecycleTokenGenerator{tokens: []string{"reset-1"}}, forgotMail, newMemoryLimiter(), policy)
	forgotCtx := requestctx.WithClientIP(context.Background(), "203.0.113.4")
	for i := 0; i < 2; i++ {
		if err := forgot.Execute(forgotCtx, dto.ForgotPasswordRequest{Email: user.Email}); err != nil {
			t.Fatalf("forgot %d error=%v", i, err)
		}
	}
	if forgotMail.forgot != 1 {
		t.Fatalf("reset emails=%d want 1", forgotMail.forgot)
	}
}

func TestLoginFailsClosedWhenTrustedClientIPMetadataIsMissing(t *testing.T) {
	uc := NewLoginUseCaseWithAbuse(&sessionUserRepository{user: testSessionUser()}, sessionPasswordEncoder{}, &sessionJWT{}, &sessionIATStore{}, newMemoryLimiter(), config.AuthAbuseConfig{LoginIPLimit: 20, LoginIdentifierLimit: 5, LoginWindow: time.Minute})
	_, err := uc.Execute(context.Background(), dto.LoginRequest{Identifier: "session-user", Password: "correct"})
	var appErr *response.AppError
	if !errors.As(err, &appErr) || appErr.Code != response.CodeServiceUnavailable {
		t.Fatalf("error=%v, want unavailable protection error", err)
	}
}

func TestEmailFlowsFailClosedWithoutTrustedClientIPMetadata(t *testing.T) {
	user := newVerifyTestUser(t, "missing-ip", false)
	users := &verifyUserRepository{users: map[string]*entity.User{user.ID: user}}
	policy := config.AuthAbuseConfig{MailIPHourlyLimit: 10, MailIPDailyLimit: 10, ResendAccountHourlyLimit: 5, ResendAccountDailyLimit: 12, ForgotAccountHourlyLimit: 3, ForgotAccountDailyLimit: 10, EmailCooldown: time.Minute}

	resendMail := &countingMail{}
	resend := NewResendVerificationUseCaseWithAbuse(users, resendMail, &lifecycleTokenGenerator{tokens: []string{"verify"}}, newLifecycleTokenRepository(), newMemoryLimiter(), policy)
	if err := resend.Execute(context.Background(), dto.ResendVerificationRequest{Email: user.Email}); err != nil || resendMail.verify != 0 {
		t.Fatalf("resend error=%v emails=%d", err, resendMail.verify)
	}

	forgotMail := &countingMail{}
	forgot := NewForgotPasswordUseCaseWithAbuse(users, newLifecycleTokenRepository(), &lifecycleTokenGenerator{tokens: []string{"reset"}}, forgotMail, newMemoryLimiter(), policy)
	if err := forgot.Execute(context.Background(), dto.ForgotPasswordRequest{Email: user.Email}); err != nil || forgotMail.forgot != 0 {
		t.Fatalf("forgot error=%v emails=%d", err, forgotMail.forgot)
	}
}

type memoryLimiter struct {
	mu     sync.Mutex
	values map[string]int
}

func newMemoryLimiter() *memoryLimiter { return &memoryLimiter{values: map[string]int{}} }
func (m *memoryLimiter) Allow(_ context.Context, purpose, scope string, limit int, _ time.Duration) (bool, time.Duration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := purpose + ":" + scope
	m.values[key]++
	return m.values[key] <= limit, time.Minute, nil
}
func (m *memoryLimiter) Reset(_ context.Context, purpose, scope string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.values, purpose+":"+scope)
	return nil
}

var _ outbound.AuthAbuseLimiter = (*memoryLimiter)(nil)

type countingMail struct{ verify, forgot int }

func (m *countingMail) SendVerificationEmail(context.Context, string, string) error {
	m.verify++
	return nil
}
func (m *countingMail) SendForgotPasswordEmail(context.Context, string, string) error {
	m.forgot++
	return nil
}

var _ outbound.MailProvider = (*countingMail)(nil)
