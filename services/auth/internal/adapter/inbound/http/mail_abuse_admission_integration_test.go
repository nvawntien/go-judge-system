package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-judge-system/pkg/config"
	"go-judge-system/pkg/response"
	authhandler "go-judge-system/services/auth/internal/adapter/inbound/http/handler/auth"
	redisadapter "go-judge-system/services/auth/internal/adapter/outbound/cache/redis"
	"go-judge-system/services/auth/internal/application/port/outbound"
	authusecase "go-judge-system/services/auth/internal/application/usecase/auth"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"
	"go-judge-system/services/auth/internal/domain/valueobject"

	"github.com/gin-gonic/gin"
	redis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Exercises Gin JSON binding, the trusted request metadata middleware, the
// production use cases, and the Redis Lua admission script. It requires an
// explicitly dedicated, non-zero Redis DB.
func TestMailAbuseAdmissionHTTPIntegration(t *testing.T) {
	url := os.Getenv("AUTH_LOGIN_INTEGRATION_REDIS_URL")
	if url == "" {
		t.Skip("set AUTH_LOGIN_INTEGRATION_REDIS_URL to a dedicated Redis DB")
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		t.Fatal(err)
	}
	if opts.DB == 0 {
		t.Fatal("AUTH_LOGIN_INTEGRATION_REDIS_URL must select a dedicated non-zero Redis DB")
	}
	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("real Redis ping: %v", err)
	}
	defer client.Close()

	t.Run("register_pair_broad_shared_nat_and_denial_side_effects", func(t *testing.T) {
		policy := mailHTTPPolicy()
		policy.RegisterIPEmailLimit, policy.RegisterBroadIPHourlyLimit = 2, 10
		h := newMailHTTPHarness(t, client, policy)
		first := h.register("10.0.0.1", "pair-user", "pair@example.test")
		h.status(t, first, http.StatusCreated)
		h.status(t, h.register("10.0.0.1", "pair-user", "pair@example.test"), http.StatusConflict)
		blocked := h.register("10.0.0.1", "pair-user", "pair@example.test")
		h.status(t, blocked, http.StatusTooManyRequests)
		if blocked.Header().Get("Retry-After") == "" {
			t.Fatal("register 429 missing Retry-After")
		}
		if h.users.created.Load() != 1 || h.tokens.saves.Load() != 1 || h.mail.verification.Load() != 1 {
			t.Fatalf("created=%d saves=%d mails=%d", h.users.created.Load(), h.tokens.saves.Load(), h.mail.verification.Load())
		}

		policy.RegisterIPEmailLimit, policy.RegisterBroadIPHourlyLimit = 3, 2
		h = newMailHTTPHarness(t, client, policy)
		for i := 0; i < 2; i++ {
			h.status(t, h.register("10.0.0.2", fmt.Sprintf("broad-%d", i), fmt.Sprintf("broad-%d@example.test", i)), http.StatusCreated)
		}
		h.status(t, h.register("10.0.0.2", "broad-2", "broad-2@example.test"), http.StatusTooManyRequests)

		policy.RegisterBroadIPHourlyLimit = 10
		h = newMailHTTPHarness(t, client, policy)
		for i := 0; i < 3; i++ {
			h.status(t, h.register("10.0.0.3", fmt.Sprintf("nat-%d", i), fmt.Sprintf("nat-%d@example.test", i)), http.StatusCreated)
		}
	})

	for _, tt := range []struct {
		name string
		path string
	}{
		{name: "resend", path: "/api/v1/auth/email/resend-verification"},
		{name: "forgot", path: "/api/v1/auth/password/forgot"},
	} {
		t.Run(tt.name+"_concurrency_generic_parity_token_safety_and_ttl", func(t *testing.T) {
			h := newMailHTTPHarness(t, client, mailHTTPPolicy())
			inactive := h.addUser("inactive", false)
			active := h.addUser("active", true)
			statuses := h.concurrentPost(tt.path, "10.0.1.1", inactive.Email, 20)
			for _, status := range statuses {
				if status != http.StatusOK {
					t.Fatalf("status=%d", status)
				}
			}
			if h.mail.total() != 1 || h.tokens.saves.Load() != 1 {
				t.Fatalf("mails=%d saves=%d", h.mail.total(), h.tokens.saves.Load())
			}
			latest := h.tokens.latest(h.namePurpose(tt.name), inactive.ID)
			if latest == "" {
				t.Fatal("missing admitted token")
			}
			h.status(t, h.post(tt.path, "10.0.1.1", map[string]string{"email": inactive.Email}), http.StatusOK)
			if h.tokens.latest(h.namePurpose(tt.name), inactive.ID) != latest || h.tokens.saves.Load() != 1 {
				t.Fatal("suppressed request replaced token")
			}

			unknown := h.post(tt.path, "10.0.1.2", map[string]string{"email": "unknown@example.test"})
			comparison := h.post(tt.path, "10.0.1.3", map[string]string{"email": "another-unknown@example.test"})
			if tt.name == "resend" {
				comparison = h.post(tt.path, "10.0.1.3", map[string]string{"email": active.Email})
			}
			if unknown.Code != comparison.Code || apiCode(t, unknown) != apiCode(t, comparison) || apiMessage(t, unknown) != apiMessage(t, comparison) {
				t.Fatalf("unknown=%s comparison=%s", unknown.Body.String(), comparison.Body.String())
			}
			if h.mail.total() != 1 {
				t.Fatalf("unexpected mail count=%d", h.mail.total())
			}
			h.assertPrivacyAndTTL(t, inactive.Email, "10.0.1.1")
		})
	}

	t.Run("missing_trusted_client_ip_fails_closed_without_mail", func(t *testing.T) {
		h := newMailHTTPHarness(t, client, mailHTTPPolicy())
		inactive := h.addUser("missing-ip", false)
		h.status(t, h.register("", "missing-ip-register", "missing-ip-register@example.test"), http.StatusServiceUnavailable)
		h.status(t, h.post("/api/v1/auth/email/resend-verification", "", map[string]string{"email": inactive.Email}), http.StatusOK)
		h.status(t, h.post("/api/v1/auth/password/forgot", "", map[string]string{"email": inactive.Email}), http.StatusOK)
		if h.users.created.Load() != 0 || h.tokens.saves.Load() != 0 || h.mail.total() != 0 {
			t.Fatalf("created=%d saves=%d mails=%d", h.users.created.Load(), h.tokens.saves.Load(), h.mail.total())
		}
	})
}

type mailHTTPHarness struct {
	client *redis.Client
	engine http.Handler
	users  *mailHTTPUsers
	tokens *mailHTTPTokens
	mail   *mailHTTPMail
}

func newMailHTTPHarness(t *testing.T, client *redis.Client, policy config.AuthAbuseConfig) *mailHTTPHarness {
	t.Helper()
	if err := client.FlushDB(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	h := &mailHTTPHarness{client: client, users: newMailHTTPUsers(), tokens: newMailHTTPTokens(), mail: &mailHTTPMail{}}
	admission := redisadapter.NewAbuseAdmission(client)
	generator := &mailHTTPGenerator{}
	register := authusecase.NewRegisterUseCaseWithAbuse(h.users, h.mail, generator, h.tokens, mailHTTPPassword{}, admission, policy)
	resend := authusecase.NewResendVerificationUseCaseWithAbuse(h.users, h.mail, generator, h.tokens, admission, policy)
	forgot := authusecase.NewForgotPasswordUseCaseWithAbuse(h.users, h.tokens, generator, h.mail, admission, policy)
	r := NewRouter(nil, nil, nil, func(c *gin.Context) { c.Next() }, zap.NewNop())
	r.engine.POST("/api/v1/auth/register", authhandler.NewRegisterHandler(register).Handle)
	r.engine.POST("/api/v1/auth/email/resend-verification", authhandler.NewResendVerificationHandler(resend).Handle)
	r.engine.POST("/api/v1/auth/password/forgot", authhandler.NewForgotPasswordHandler(forgot).Handle)
	h.engine = r.engine
	return h
}

func (h *mailHTTPHarness) addUser(name string, active bool) *entity.User {
	email, _ := valueobject.NewEmail(name + "@example.test")
	user := entity.NewUser("Test User", name, email, valueobject.NewPasswordFromHash("hash"))
	user.ID, user.IsActive = name, active
	h.users.add(user)
	return user
}
func (h *mailHTTPHarness) register(ip, username, email string) *httptest.ResponseRecorder {
	return h.post("/api/v1/auth/register", ip, map[string]string{"full_name": "Test User", "username": username, "email": email, "password": "StrongPass123!"})
}
func (h *mailHTTPHarness) post(path, ip string, payload map[string]string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if ip != "" {
		req.Header.Set("X-Client-IP", ip)
	}
	out := httptest.NewRecorder()
	h.engine.ServeHTTP(out, req)
	return out
}
func (h *mailHTTPHarness) concurrentPost(path, ip, email string, total int) []int {
	start := make(chan struct{})
	results := make(chan int, total)
	var wg sync.WaitGroup
	for range total {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; results <- h.post(path, ip, map[string]string{"email": email}).Code }()
	}
	close(start)
	wg.Wait()
	close(results)
	statuses := make([]int, 0, total)
	for status := range results {
		statuses = append(statuses, status)
	}
	return statuses
}
func (h *mailHTTPHarness) status(t *testing.T, out *httptest.ResponseRecorder, want int) {
	t.Helper()
	if out.Code != want {
		t.Fatalf("status=%d want=%d body=%s", out.Code, want, out.Body.String())
	}
}
func (h *mailHTTPHarness) assertPrivacyAndTTL(t *testing.T, sensitive ...string) {
	t.Helper()
	keys, err := h.client.Keys(context.Background(), "auth:abuse:*").Result()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) == 0 {
		t.Fatal("expected abuse keys")
	}
	for _, key := range keys {
		for _, raw := range sensitive {
			if strings.Contains(key, raw) {
				t.Fatalf("key exposes sensitive scope: %q", key)
			}
		}
		ttl, err := h.client.PTTL(context.Background(), key).Result()
		if err != nil || ttl <= 0 {
			t.Fatalf("key=%q ttl=%s err=%v", key, ttl, err)
		}
	}
}
func (h *mailHTTPHarness) namePurpose(name string) outbound.TokenPurpose {
	if name == "resend" {
		return outbound.TokenPurposeVerifyEmail
	}
	return outbound.TokenPurposeResetPassword
}

func mailHTTPPolicy() config.AuthAbuseConfig {
	return config.AuthAbuseConfig{RegisterIPEmailLimit: 3, RegisterBroadIPHourlyLimit: 100, RegisterBroadIPDailyLimit: 500, ResendAccountHourlyLimit: 5, ResendAccountDailyLimit: 12, ResendIPAccountLimit: 5, ResendBroadIPHourlyLimit: 100, ResendBroadIPDailyLimit: 500, ForgotAccountHourlyLimit: 3, ForgotAccountDailyLimit: 10, ForgotIPAccountLimit: 5, ForgotBroadIPHourlyLimit: 100, ForgotBroadIPDailyLimit: 500, MailHourlyWindow: time.Minute, MailDailyWindow: time.Hour, EmailCooldown: time.Hour}
}

func apiCode(t *testing.T, out *httptest.ResponseRecorder) int {
	t.Helper()
	var body response.APIResponse
	if err := json.Unmarshal(out.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Code
}
func apiMessage(t *testing.T, out *httptest.ResponseRecorder) string {
	t.Helper()
	var body response.APIResponse
	if err := json.Unmarshal(out.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Msg
}

type mailHTTPUsers struct {
	mu      sync.RWMutex
	byEmail map[string]*entity.User
	byName  map[string]*entity.User
	created atomic.Int32
}

func newMailHTTPUsers() *mailHTTPUsers {
	return &mailHTTPUsers{byEmail: map[string]*entity.User{}, byName: map[string]*entity.User{}}
}
func (r *mailHTTPUsers) add(user *entity.User) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byEmail[user.Email], r.byName[user.Username] = user, user
}
func (r *mailHTTPUsers) CreateUser(_ context.Context, user *entity.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byEmail[user.Email]; ok {
		return domain.ErrDuplicateEntry
	}
	r.byEmail[user.Email], r.byName[user.Username] = user, user
	r.created.Add(1)
	return nil
}
func (r *mailHTTPUsers) GetUserByEmail(_ context.Context, email string) (*entity.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.byEmail[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}
func (r *mailHTTPUsers) GetUserByUsername(_ context.Context, username string) (*entity.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.byName[username]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}
func (r *mailHTTPUsers) GetUserById(context.Context, string) (*entity.User, error) {
	return nil, domain.ErrUserNotFound
}
func (r *mailHTTPUsers) ListUsers(context.Context, outbound.ListUsersFilter) (outbound.ListUsersResult, error) {
	return outbound.ListUsersResult{}, nil
}
func (r *mailHTTPUsers) SearchPublicUsers(context.Context, outbound.SearchPublicUsersFilter) (outbound.SearchPublicUsersResult, error) {
	return outbound.SearchPublicUsersResult{}, nil
}
func (r *mailHTTPUsers) UpdateUser(context.Context, *entity.User) error                  { return nil }
func (r *mailHTTPUsers) UpdatePassword(context.Context, string, string, time.Time) error { return nil }
func (r *mailHTTPUsers) UpdateProfile(context.Context, string, outbound.ProfileUpdates) error {
	return nil
}
func (r *mailHTTPUsers) UpdateAvatar(context.Context, string, string, string, time.Time) error {
	return nil
}
func (r *mailHTTPUsers) DeleteUser(context.Context, string) error { return nil }

type mailHTTPTokens struct {
	mu              sync.Mutex
	values          map[outbound.TokenPurpose]map[string]string
	latestByPurpose map[outbound.TokenPurpose]map[string]string
	saves           atomic.Int32
}

func newMailHTTPTokens() *mailHTTPTokens {
	return &mailHTTPTokens{values: map[outbound.TokenPurpose]map[string]string{}, latestByPurpose: map[outbound.TokenPurpose]map[string]string{}}
}
func (r *mailHTTPTokens) Save(_ context.Context, purpose outbound.TokenPurpose, token, id string, _ time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.values[purpose] == nil {
		r.values[purpose], r.latestByPurpose[purpose] = map[string]string{}, map[string]string{}
	}
	r.values[purpose][token], r.latestByPurpose[purpose][id] = id, token
	r.saves.Add(1)
	return nil
}
func (r *mailHTTPTokens) FindByToken(_ context.Context, purpose outbound.TokenPurpose, token string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.values[purpose][token]
	if !ok || r.latestByPurpose[purpose][id] != token {
		return "", domain.ErrInvalidOrExpiredToken
	}
	return id, nil
}
func (r *mailHTTPTokens) Consume(ctx context.Context, purpose outbound.TokenPurpose, token string) (string, error) {
	return r.FindByToken(ctx, purpose, token)
}
func (r *mailHTTPTokens) Delete(context.Context, outbound.TokenPurpose, string) error { return nil }
func (r *mailHTTPTokens) TryAcquireResendCooldown(context.Context, outbound.TokenPurpose, string, time.Duration) (bool, error) {
	return true, nil
}
func (r *mailHTTPTokens) latest(purpose outbound.TokenPurpose, id string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.latestByPurpose[purpose][id]
}

type mailHTTPGenerator struct{ sequence atomic.Int32 }

func (g *mailHTTPGenerator) Generate(string) string {
	return fmt.Sprintf("token-%d", g.sequence.Add(1))
}
func (g *mailHTTPGenerator) Hash(token string) string { return "hash:" + token }

type mailHTTPPassword struct{}

func (mailHTTPPassword) HashAndSalt([]byte) (string, error)   { return "hash", nil }
func (mailHTTPPassword) ComparePasswords(string, []byte) bool { return false }

type mailHTTPMail struct{ verification, forgot atomic.Int32 }

func (m *mailHTTPMail) SendVerificationEmail(context.Context, string, string) error {
	m.verification.Add(1)
	return nil
}
func (m *mailHTTPMail) SendForgotPasswordEmail(context.Context, string, string) error {
	m.forgot.Add(1)
	return nil
}
func (m *mailHTTPMail) total() int32 { return m.verification.Load() + m.forgot.Load() }

var _ outbound.UserRepository = (*mailHTTPUsers)(nil)
var _ outbound.TokenRepository = (*mailHTTPTokens)(nil)
var _ outbound.TokenGenerator = (*mailHTTPGenerator)(nil)
var _ outbound.MailProvider = (*mailHTTPMail)(nil)
var _ outbound.PasswordEncoder = mailHTTPPassword{}
