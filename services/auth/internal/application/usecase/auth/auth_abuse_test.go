package auth

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-judge-system/pkg/config"
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
	for i := 0; i < 2; i++ {
		_, err := uc.Execute(context.Background(), dto.LoginRequest{Identifier: " SESSION-USER ", Password: "bad", ClientIP: "203.0.113.1"})
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("attempt %d error=%v", i, err)
		}
	}
	_, err := uc.Execute(context.Background(), dto.LoginRequest{Identifier: "session-user", Password: "bad", ClientIP: "203.0.113.1"})
	if !errors.Is(err, domain.ErrRateLimitExceeded) {
		t.Fatalf("blocked error=%v", err)
	}
	// A new identifier bucket is independent; a successful login clears its
	// failed-attempt state so the next failure is not throttled.
	_, err = uc.Execute(context.Background(), dto.LoginRequest{Identifier: "other", Password: "correct", ClientIP: "203.0.113.1"})
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
	for i := 0; i < 2; i++ {
		if err := resend.Execute(context.Background(), dto.ResendVerificationRequest{Email: user.Email, ClientIP: "203.0.113.2"}); err != nil {
			t.Fatalf("resend %d error=%v", i, err)
		}
	}
	if mail.verify != 1 {
		t.Fatalf("verification emails=%d want 1", mail.verify)
	}
	if err := resend.Execute(context.Background(), dto.ResendVerificationRequest{Email: "missing@example.test", ClientIP: "203.0.113.3"}); err != nil || mail.verify != 1 {
		t.Fatalf("unknown resend error=%v emails=%d", err, mail.verify)
	}

	forgotMail := &countingMail{}
	forgot := NewForgotPasswordUseCaseWithAbuse(users, newLifecycleTokenRepository(), &lifecycleTokenGenerator{tokens: []string{"reset-1"}}, forgotMail, newMemoryLimiter(), policy)
	for i := 0; i < 2; i++ {
		if err := forgot.Execute(context.Background(), dto.ForgotPasswordRequest{Email: user.Email, ClientIP: "203.0.113.4"}); err != nil {
			t.Fatalf("forgot %d error=%v", i, err)
		}
	}
	if forgotMail.forgot != 1 {
		t.Fatalf("reset emails=%d want 1", forgotMail.forgot)
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
