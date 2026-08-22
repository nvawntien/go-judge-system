package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
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

func TestLoginPairThrottleDoesNotLockVictimFromDifferentIP(t *testing.T) {
	limiter := newMemoryLimiter()
	policy := loginPolicy()
	user := testSessionUser()
	uc := NewLoginUseCaseWithAbuse(&sessionUserRepository{user: user}, sessionPasswordEncoder{}, &sessionJWT{}, &sessionIATStore{}, limiter, policy)
	attackerCtx := requestctx.WithClientIP(context.Background(), "203.0.113.1")
	for i := 0; i < policy.LoginIPIdentifierLimit; i++ {
		_, err := uc.Execute(attackerCtx, dto.LoginRequest{Identifier: " SESSION-USER ", Password: "bad"})
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("attempt %d error=%v", i, err)
		}
	}
	_, err := uc.Execute(attackerCtx, dto.LoginRequest{Identifier: "session-user", Password: "bad"})
	if !errors.Is(err, domain.ErrRateLimitExceeded) {
		t.Fatalf("blocked error=%v", err)
	}

	_, err = uc.Execute(requestctx.WithClientIP(context.Background(), "203.0.113.2"), dto.LoginRequest{Identifier: "session-user", Password: "correct"})
	if err != nil {
		t.Fatalf("victim login from a different IP=%v", err)
	}
}

func TestLoginIdentifierRiskDoesNotHardLock(t *testing.T) {
	limiter := newMemoryLimiter()
	policy := loginPolicy()
	policy.LoginIdentifierRiskLimit = 2
	uc := NewLoginUseCaseWithAbuse(&sessionUserRepository{user: testSessionUser()}, sessionPasswordEncoder{}, &sessionJWT{}, &sessionIATStore{}, limiter, policy)
	for i := 0; i < 6; i++ {
		ctx := requestctx.WithClientIP(context.Background(), "203.0.113."+strconv.Itoa(i+10))
		_, err := uc.Execute(ctx, dto.LoginRequest{Identifier: "session-user", Password: "bad"})
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("failure %d=%v", i, err)
		}
	}
	_, err := uc.Execute(requestctx.WithClientIP(context.Background(), "203.0.113.99"), dto.LoginRequest{Identifier: "session-user", Password: "correct"})
	if err != nil {
		t.Fatalf("identifier risk hard-locked correct login: %v", err)
	}
}

func TestLoginIdentifierRiskAddsBoundedDelayWithoutBlockingSuccess(t *testing.T) {
	limiter := newMemoryLimiter()
	policy := loginPolicy()
	policy.LoginIdentifierRiskLimit = 1
	policy.LoginIdentifierRiskDelay = 10 * time.Millisecond
	uc := NewLoginUseCaseWithAbuse(&sessionUserRepository{user: testSessionUser()}, sessionPasswordEncoder{}, &sessionJWT{}, &sessionIATStore{}, limiter, policy)
	if _, err := uc.Execute(requestctx.WithClientIP(context.Background(), "203.0.113.20"), dto.LoginRequest{Identifier: "session-user", Password: "bad"}); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err := uc.Execute(requestctx.WithClientIP(context.Background(), "203.0.113.21"), dto.LoginRequest{Identifier: "session-user", Password: "correct"}); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed < 8*time.Millisecond {
		t.Fatalf("risk delay=%s, want bounded delay", elapsed)
	}
}

func TestLoginBroadIPFailureLimitCountsOnlyFailures(t *testing.T) {
	limiter := newMemoryLimiter()
	policy := loginPolicy()
	policy.LoginBroadIPLimit = 3
	uc := NewLoginUseCaseWithAbuse(&sessionUserRepository{user: testSessionUser()}, sessionPasswordEncoder{}, &sessionJWT{}, &sessionIATStore{}, limiter, policy)
	ctx := requestctx.WithClientIP(context.Background(), "203.0.113.50")
	for i := 0; i < policy.LoginBroadIPLimit; i++ {
		_, err := uc.Execute(ctx, dto.LoginRequest{Identifier: fmt.Sprintf("target-%d", i), Password: "bad"})
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("failure %d=%v", i, err)
		}
	}
	_, err := uc.Execute(ctx, dto.LoginRequest{Identifier: "target-next", Password: "bad"})
	if !errors.Is(err, domain.ErrRateLimitExceeded) {
		t.Fatalf("broad IP error=%v", err)
	}

	legitimateLimiter := newMemoryLimiter()
	legitimate := NewLoginUseCaseWithAbuse(&sessionUserRepository{user: testSessionUser()}, sessionPasswordEncoder{}, &sessionJWT{}, &sessionIATStore{}, legitimateLimiter, policy)
	for i := 0; i < 25; i++ {
		_, err := legitimate.Execute(ctx, dto.LoginRequest{Identifier: fmt.Sprintf("student-%d", i), Password: "correct"})
		if err != nil {
			t.Fatalf("shared NAT success %d=%v", i, err)
		}
	}
	count, _, err := legitimateLimiter.Count(context.Background(), "login:ip", "203.0.113.50")
	if err != nil || count != 0 {
		t.Fatalf("successful logins broad failures=%d err=%v", count, err)
	}
}

func TestLoginSharedNATOccasionalFailuresStayBelowBroadLimit(t *testing.T) {
	uc := NewLoginUseCaseWithAbuse(&sessionUserRepository{user: testSessionUser()}, sessionPasswordEncoder{}, &sessionJWT{}, &sessionIATStore{}, newMemoryLimiter(), loginPolicy())
	ctx := requestctx.WithClientIP(context.Background(), "203.0.113.55")
	for i := 0; i < 25; i++ {
		_, err := uc.Execute(ctx, dto.LoginRequest{Identifier: fmt.Sprintf("student-%d", i), Password: "bad"})
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("failure %d=%v", i, err)
		}
	}
}

func TestLoginSuccessClearsPairAndIdentifierButNotBroadIP(t *testing.T) {
	limiter := newMemoryLimiter()
	uc := NewLoginUseCaseWithAbuse(&sessionUserRepository{user: testSessionUser()}, sessionPasswordEncoder{}, &sessionJWT{}, &sessionIATStore{}, limiter, loginPolicy())
	ctx := requestctx.WithClientIP(context.Background(), "203.0.113.60")
	if _, err := uc.Execute(ctx, dto.LoginRequest{Identifier: "session-user", Password: "bad"}); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatal(err)
	}
	if _, err := uc.Execute(ctx, dto.LoginRequest{Identifier: "session-user", Password: "correct"}); err != nil {
		t.Fatal(err)
	}
	for _, scope := range []struct{ purpose, scope string }{{"login:ip-identifier", loginIPIdentifierScope("203.0.113.60", "session-user")}, {"login:identifier", "session-user"}} {
		count, _, err := limiter.Count(context.Background(), scope.purpose, scope.scope)
		if err != nil || count != 0 {
			t.Fatalf("%s count=%d err=%v", scope.purpose, count, err)
		}
	}
	count, _, err := limiter.Count(context.Background(), "login:ip", "203.0.113.60")
	if err != nil || count != 1 {
		t.Fatalf("broad IP count=%d err=%v", count, err)
	}
}

func TestLoginUnknownIdentifierUsesSameFailureResponseAndScopes(t *testing.T) {
	policy := loginPolicy()
	ctx := requestctx.WithClientIP(context.Background(), "203.0.113.70")
	knownLimiter := newMemoryLimiter()
	known := NewLoginUseCaseWithAbuse(&sessionUserRepository{user: testSessionUser()}, sessionPasswordEncoder{}, &sessionJWT{}, &sessionIATStore{}, knownLimiter, policy)
	_, knownErr := known.Execute(ctx, dto.LoginRequest{Identifier: "target", Password: "bad"})
	unknownLimiter := newMemoryLimiter()
	unknown := NewLoginUseCaseWithAbuse(&sessionUserRepository{err: domain.ErrUserNotFound}, sessionPasswordEncoder{}, &sessionJWT{}, &sessionIATStore{}, unknownLimiter, policy)
	_, unknownErr := unknown.Execute(ctx, dto.LoginRequest{Identifier: "target", Password: "bad"})
	if !errors.Is(knownErr, domain.ErrInvalidCredentials) || !errors.Is(unknownErr, domain.ErrInvalidCredentials) {
		t.Fatalf("known=%v unknown=%v", knownErr, unknownErr)
	}
	for _, scope := range []struct{ purpose, scope string }{{"login:ip-identifier", loginIPIdentifierScope("203.0.113.70", "target")}, {"login:identifier", "target"}, {"login:ip", "203.0.113.70"}} {
		knownCount, _, _ := knownLimiter.Count(context.Background(), scope.purpose, scope.scope)
		unknownCount, _, _ := unknownLimiter.Count(context.Background(), scope.purpose, scope.scope)
		if knownCount != 1 || unknownCount != 1 {
			t.Fatalf("%s known=%d unknown=%d", scope.purpose, knownCount, unknownCount)
		}
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
	uc := NewLoginUseCaseWithAbuse(&sessionUserRepository{user: testSessionUser()}, sessionPasswordEncoder{}, &sessionJWT{}, &sessionIATStore{}, newMemoryLimiter(), loginPolicy())
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
func (m *memoryLimiter) Count(_ context.Context, purpose, scope string) (int64, time.Duration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return int64(m.values[purpose+":"+scope]), time.Minute, nil
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

func loginPolicy() config.AuthAbuseConfig {
	return config.AuthAbuseConfig{LoginIPIdentifierLimit: 5, LoginIdentifierRiskLimit: 5, LoginIdentifierRiskDelay: 0, LoginBroadIPLimit: 200, LoginWindow: time.Minute}
}
