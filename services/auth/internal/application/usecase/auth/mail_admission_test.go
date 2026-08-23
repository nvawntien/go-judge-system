package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-judge-system/pkg/requestctx"
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"
)

func TestMailAdmissionDeniedRequestsHaveNoSideEffectsOrReplaceTokens(t *testing.T) {
	ctx := requestctx.WithClientIP(context.Background(), "203.0.113.1")
	policy := mailAbusePolicy()
	user := newVerifyTestUser(t, "mail-admission", false)
	users := newMailAdmissionUsers(user)

	t.Run("register", func(t *testing.T) {
		mail := &synchronizedCountingMail{}
		tokens := &countingTokenRepository{lifecycleTokenRepository: newLifecycleTokenRepository()}
		uc := NewRegisterUseCaseWithAbuse(users, mail, &safeTokenGenerator{}, tokens, lifecyclePasswordEncoder{}, deniedAdmission{}, policy)
		err := uc.Execute(ctx, dto.RegisterRequest{FullName: "New User", Username: "new-user", Email: "new-user@example.test", Password: "StrongPass123!"})
		var appErr *response.AppError
		if !errors.As(err, &appErr) || appErr.Code != response.CodeRateLimitExceeded {
			t.Fatalf("error=%v", err)
		}
		if users.created.Load() != 0 || tokens.saves.Load() != 0 || mail.verification.Load() != 0 {
			t.Fatalf("created=%d token saves=%d emails=%d", users.created.Load(), tokens.saves.Load(), mail.verification.Load())
		}
	})

	for _, tt := range []struct {
		name    string
		purpose outbound.TokenPurpose
		execute func(outbound.TokenRepository, outbound.TokenGenerator, outbound.MailProvider) error
	}{
		{
			name: "resend", purpose: outbound.TokenPurposeVerifyEmail,
			execute: func(tokens outbound.TokenRepository, generator outbound.TokenGenerator, mail outbound.MailProvider) error {
				return NewResendVerificationUseCaseWithAbuse(users, mail, generator, tokens, deniedAdmission{}, policy).Execute(ctx, dto.ResendVerificationRequest{Email: user.Email})
			},
		},
		{
			name: "forgot", purpose: outbound.TokenPurposeResetPassword,
			execute: func(tokens outbound.TokenRepository, generator outbound.TokenGenerator, mail outbound.MailProvider) error {
				return NewForgotPasswordUseCaseWithAbuse(users, tokens, generator, mail, deniedAdmission{}, policy).Execute(ctx, dto.ForgotPasswordRequest{Email: user.Email})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tokens := &countingTokenRepository{lifecycleTokenRepository: newLifecycleTokenRepository()}
			if err := tokens.Save(ctx, tt.purpose, "old-token", user.ID, time.Hour); err != nil {
				t.Fatal(err)
			}
			generator := &safeTokenGenerator{}
			mail := &synchronizedCountingMail{}
			if err := tt.execute(tokens, generator, mail); err != nil {
				t.Fatalf("generic suppressed request error=%v", err)
			}
			if _, err := tokens.FindByToken(ctx, tt.purpose, "old-token"); err != nil {
				t.Fatalf("denied request replaced old token: %v", err)
			}
			if generator.generated.Load() != 0 || tokens.saves.Load() != 1 || mail.total() != 0 {
				t.Fatalf("generated=%d saves=%d mails=%d", generator.generated.Load(), tokens.saves.Load(), mail.total())
			}
		})
	}
}

func TestMailAdmissionConcurrencyBoundsMailAndRegisterSideEffects(t *testing.T) {
	policy := mailAbusePolicy()
	policy.EmailCooldown = time.Hour
	ctx := requestctx.WithClientIP(context.Background(), "203.0.113.2")
	user := newVerifyTestUser(t, "concurrent-mail", false)

	for _, tt := range []struct {
		name string
		run  func(*synchronizedCountingMail, *countingTokenRepository, outbound.AbuseAdmission) error
	}{
		{
			name: "resend",
			run: func(mail *synchronizedCountingMail, tokens *countingTokenRepository, admission outbound.AbuseAdmission) error {
				return NewResendVerificationUseCaseWithAbuse(newMailAdmissionUsers(user), mail, &safeTokenGenerator{}, tokens, admission, policy).Execute(ctx, dto.ResendVerificationRequest{Email: user.Email})
			},
		},
		{
			name: "forgot",
			run: func(mail *synchronizedCountingMail, tokens *countingTokenRepository, admission outbound.AbuseAdmission) error {
				return NewForgotPasswordUseCaseWithAbuse(newMailAdmissionUsers(user), tokens, &safeTokenGenerator{}, mail, admission, policy).Execute(ctx, dto.ForgotPasswordRequest{Email: user.Email})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mail := &synchronizedCountingMail{}
			tokens := &countingTokenRepository{lifecycleTokenRepository: newLifecycleTokenRepository()}
			admission := newMemoryAdmission()
			runConcurrent(t, 20, func() error { return tt.run(mail, tokens, admission) })
			if mails := mail.total(); mails != 1 {
				t.Fatalf("emails=%d want 1", mails)
			}
			if saves := tokens.saves.Load(); saves != 1 {
				t.Fatalf("token saves=%d want 1", saves)
			}
		})
	}

	t.Run("register", func(t *testing.T) {
		mail := &synchronizedCountingMail{}
		tokens := &countingTokenRepository{lifecycleTokenRepository: newLifecycleTokenRepository()}
		users := newMailAdmissionUsers()
		uc := NewRegisterUseCaseWithAbuse(users, mail, &safeTokenGenerator{}, tokens, lifecyclePasswordEncoder{}, newMemoryAdmission(), policy)
		runConcurrentAllowErrors(t, 20, func() error {
			return uc.Execute(ctx, dto.RegisterRequest{FullName: "Concurrent", Username: "concurrent-user", Email: "concurrent@example.test", Password: "StrongPass123!"})
		})
		if users.created.Load() > int32(policy.RegisterIPEmailLimit) || tokens.saves.Load() > int32(policy.RegisterIPEmailLimit) || mail.verification.Load() > int32(policy.RegisterIPEmailLimit) {
			t.Fatalf("created=%d saves=%d mails=%d", users.created.Load(), tokens.saves.Load(), mail.verification.Load())
		}
	})
}

func TestMailAdmissionMissingIPAndStoreFailureFailClosed(t *testing.T) {
	policy := mailAbusePolicy()
	user := newVerifyTestUser(t, "protection-failure", false)
	users := newMailAdmissionUsers(user)
	mail := &synchronizedCountingMail{}
	tokens := &countingTokenRepository{lifecycleTokenRepository: newLifecycleTokenRepository()}

	if err := NewResendVerificationUseCaseWithAbuse(users, mail, &safeTokenGenerator{}, tokens, failingAdmission{}, policy).Execute(requestctx.WithClientIP(context.Background(), "203.0.113.3"), dto.ResendVerificationRequest{Email: user.Email}); err != nil || mail.total() != 0 || tokens.saves.Load() != 0 {
		t.Fatalf("resend error=%v mails=%d saves=%d", err, mail.total(), tokens.saves.Load())
	}
	if err := NewForgotPasswordUseCaseWithAbuse(users, tokens, &safeTokenGenerator{}, mail, failingAdmission{}, policy).Execute(requestctx.WithClientIP(context.Background(), "203.0.113.3"), dto.ForgotPasswordRequest{Email: user.Email}); err != nil || mail.total() != 0 || tokens.saves.Load() != 0 {
		t.Fatalf("forgot error=%v mails=%d saves=%d", err, mail.total(), tokens.saves.Load())
	}

	err := NewRegisterUseCaseWithAbuse(newMailAdmissionUsers(), mail, &safeTokenGenerator{}, tokens, lifecyclePasswordEncoder{}, failingAdmission{}, policy).Execute(requestctx.WithClientIP(context.Background(), "203.0.113.3"), dto.RegisterRequest{FullName: "Failure", Username: "failure-user", Email: "failure@example.test", Password: "StrongPass123!"})
	var appErr *response.AppError
	if !errors.As(err, &appErr) || appErr.Code != response.CodeServiceUnavailable {
		t.Fatalf("register error=%v", err)
	}
}

func TestMailAdmissionRequestsContainEveryRequiredScope(t *testing.T) {
	ctx := requestctx.WithClientIP(context.Background(), "203.0.113.4")
	policy := mailAbusePolicy()
	assertRequest := func(t *testing.T, admission *captureAdmission, want ...string) {
		t.Helper()
		if admission.calls != 1 {
			t.Fatalf("admission calls=%d want 1", admission.calls)
		}
		got := map[string]bool{}
		for _, scope := range admission.request.Scopes {
			got[scope.Purpose] = true
		}
		for _, purpose := range want {
			if !got[purpose] {
				t.Fatalf("missing scope %q in %+v", purpose, admission.request.Scopes)
			}
		}
	}

	t.Run("register", func(t *testing.T) {
		admission := &captureAdmission{}
		users := newMailAdmissionUsers()
		uc := NewRegisterUseCaseWithAbuse(users, &synchronizedCountingMail{}, &safeTokenGenerator{}, &countingTokenRepository{lifecycleTokenRepository: newLifecycleTokenRepository()}, lifecyclePasswordEncoder{}, admission, policy)
		if err := uc.Execute(ctx, dto.RegisterRequest{FullName: "Scope User", Username: "scope-user", Email: "scope-user@example.test", Password: "StrongPass123!"}); err != nil {
			t.Fatal(err)
		}
		assertRequest(t, admission, "register:ip-email", "register:ip-hour", "register:ip-day")
	})

	for _, tt := range []struct {
		name string
		run  func(*captureAdmission, *mailAdmissionUsers, *entity.User) error
		want []string
	}{
		{name: "resend", want: []string{"resend:ip-account", "resend:account-hour", "resend:account-day", "resend:ip-hour", "resend:ip-day"}, run: func(admission *captureAdmission, users *mailAdmissionUsers, user *entity.User) error {
			return NewResendVerificationUseCaseWithAbuse(users, &synchronizedCountingMail{}, &safeTokenGenerator{}, &countingTokenRepository{lifecycleTokenRepository: newLifecycleTokenRepository()}, admission, policy).Execute(ctx, dto.ResendVerificationRequest{Email: user.Email})
		}},
		{name: "forgot", want: []string{"forgot:ip-account", "forgot:account-hour", "forgot:account-day", "forgot:ip-hour", "forgot:ip-day"}, run: func(admission *captureAdmission, users *mailAdmissionUsers, user *entity.User) error {
			return NewForgotPasswordUseCaseWithAbuse(users, &countingTokenRepository{lifecycleTokenRepository: newLifecycleTokenRepository()}, &safeTokenGenerator{}, &synchronizedCountingMail{}, admission, policy).Execute(ctx, dto.ForgotPasswordRequest{Email: user.Email})
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			admission := &captureAdmission{}
			user := newVerifyTestUser(t, "scope-"+tt.name, false)
			if err := tt.run(admission, newMailAdmissionUsers(user), user); err != nil {
				t.Fatal(err)
			}
			assertRequest(t, admission, tt.want...)
			if admission.request.Cooldown == nil || admission.request.Cooldown.Purpose != tt.name+":cooldown" {
				t.Fatalf("cooldown=%+v", admission.request.Cooldown)
			}
		})
	}
}

type deniedAdmission struct{}

func (deniedAdmission) Acquire(context.Context, outbound.AdmissionRequest) (outbound.AdmissionResult, error) {
	return outbound.AdmissionResult{RetryAfter: time.Minute}, nil
}

type failingAdmission struct{}

func (failingAdmission) Acquire(context.Context, outbound.AdmissionRequest) (outbound.AdmissionResult, error) {
	return outbound.AdmissionResult{}, errors.New("redis unavailable")
}

type captureAdmission struct {
	calls   int
	request outbound.AdmissionRequest
}

func (a *captureAdmission) Acquire(_ context.Context, request outbound.AdmissionRequest) (outbound.AdmissionResult, error) {
	a.calls++
	a.request = request
	return outbound.AdmissionResult{Allowed: true}, nil
}

type safeTokenGenerator struct{ generated atomic.Int32 }

func (g *safeTokenGenerator) Generate(string) string {
	return fmt.Sprintf("token-%d", g.generated.Add(1))
}
func (g *safeTokenGenerator) Hash(token string) string { return "hashed:" + token }

type synchronizedCountingMail struct{ verification, forgot atomic.Int32 }

func (m *synchronizedCountingMail) SendVerificationEmail(context.Context, string, string) error {
	m.verification.Add(1)
	return nil
}
func (m *synchronizedCountingMail) SendForgotPasswordEmail(context.Context, string, string) error {
	m.forgot.Add(1)
	return nil
}
func (m *synchronizedCountingMail) total() int32 { return m.verification.Load() + m.forgot.Load() }

type countingTokenRepository struct {
	*lifecycleTokenRepository
	saves atomic.Int32
}

func (r *countingTokenRepository) Save(ctx context.Context, purpose outbound.TokenPurpose, token, identifier string, ttl time.Duration) error {
	r.saves.Add(1)
	return r.lifecycleTokenRepository.Save(ctx, purpose, token, identifier, ttl)
}

type mailAdmissionUsers struct {
	mu      sync.RWMutex
	byEmail map[string]*entity.User
	byName  map[string]*entity.User
	created atomic.Int32
}

func newMailAdmissionUsers(users ...*entity.User) *mailAdmissionUsers {
	r := &mailAdmissionUsers{byEmail: map[string]*entity.User{}, byName: map[string]*entity.User{}}
	for _, user := range users {
		r.byEmail[user.Email] = user
		r.byName[user.Username] = user
	}
	return r
}

func (r *mailAdmissionUsers) CreateUser(_ context.Context, user *entity.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byEmail[user.Email]; ok {
		return domain.ErrDuplicateEntry
	}
	r.byEmail[user.Email], r.byName[user.Username] = user, user
	r.created.Add(1)
	return nil
}
func (r *mailAdmissionUsers) GetUserByEmail(_ context.Context, email string) (*entity.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.byEmail[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}
func (r *mailAdmissionUsers) GetUserByUsername(_ context.Context, username string) (*entity.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.byName[username]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}
func (r *mailAdmissionUsers) GetUserById(context.Context, string) (*entity.User, error) {
	return nil, domain.ErrUserNotFound
}
func (r *mailAdmissionUsers) ListUsers(context.Context, outbound.ListUsersFilter) (outbound.ListUsersResult, error) {
	return outbound.ListUsersResult{}, nil
}
func (r *mailAdmissionUsers) SearchPublicUsers(context.Context, outbound.SearchPublicUsersFilter) (outbound.SearchPublicUsersResult, error) {
	return outbound.SearchPublicUsersResult{}, nil
}
func (r *mailAdmissionUsers) UpdateUser(context.Context, *entity.User) error { return nil }
func (r *mailAdmissionUsers) UpdatePassword(context.Context, string, string, time.Time) error {
	return nil
}
func (r *mailAdmissionUsers) UpdateProfile(context.Context, string, outbound.ProfileUpdates) error {
	return nil
}
func (r *mailAdmissionUsers) UpdateAvatar(context.Context, string, string, string, time.Time) error {
	return nil
}
func (r *mailAdmissionUsers) DeleteUser(context.Context, string) error { return nil }

func runConcurrent(t *testing.T, total int, run func() error) {
	t.Helper()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range total {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if err := run(); err != nil {
				t.Errorf("execute: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
}

func runConcurrentAllowErrors(t *testing.T, total int, run func() error) {
	t.Helper()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range total {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; _ = run() }()
	}
	close(start)
	wg.Wait()
}

var _ outbound.AbuseAdmission = deniedAdmission{}
var _ outbound.AbuseAdmission = failingAdmission{}
var _ outbound.AbuseAdmission = (*captureAdmission)(nil)
var _ outbound.TokenGenerator = (*safeTokenGenerator)(nil)
var _ outbound.MailProvider = (*synchronizedCountingMail)(nil)
var _ outbound.TokenRepository = (*countingTokenRepository)(nil)
var _ outbound.UserRepository = (*mailAdmissionUsers)(nil)
