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
	"testing"
	"time"

	pkgauth "go-judge-system/pkg/auth"
	"go-judge-system/pkg/config"
	"go-judge-system/pkg/rbac"
	"go-judge-system/pkg/response"
	root "go-judge-system/services/auth/internal/adapter/inbound/http/handler"
	authhandler "go-judge-system/services/auth/internal/adapter/inbound/http/handler/auth"
	redisadapter "go-judge-system/services/auth/internal/adapter/outbound/cache/redis"
	"go-judge-system/services/auth/internal/application/port/outbound"
	authusecase "go-judge-system/services/auth/internal/application/usecase/auth"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"

	"github.com/gin-gonic/gin"
	redis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// These tests require an explicitly dedicated, non-zero Redis DB. Example:
// AUTH_LOGIN_INTEGRATION_REDIS_URL=redis://127.0.0.1:16379/15 \
// go test ./internal/adapter/inbound/http -run TestLoginAbuseIntegration -count=1
func TestLoginAbuseIntegration(t *testing.T) {
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

	t.Run("sequential_pair_ttl_and_key_privacy", func(t *testing.T) {
		h := newLoginIntegrationHarness(t, client, testLoginPolicy())
		for i := 0; i < 5; i++ {
			h.expectStatus(t, h.login("10.0.0.1", h.victim, "wrong"), http.StatusUnauthorized)
		}
		blocked := h.login("10.0.0.1", h.victim, "wrong")
		h.expectStatus(t, blocked, http.StatusTooManyRequests)
		if blocked.Header().Get("Retry-After") == "" {
			t.Fatal("missing Retry-After")
		}
		count, ttl, err := h.limiter.Count(context.Background(), "login:ip-identifier", "10.0.0.1\x00"+h.victim)
		if err != nil || count != 5 || ttl <= 0 || ttl > h.policy.LoginWindow {
			t.Fatalf("count=%d ttl=%s err=%v", count, ttl, err)
		}
		keys, err := client.Keys(context.Background(), "auth:abuse:*").Result()
		if err != nil {
			t.Fatal(err)
		}
		if len(keys) == 0 {
			t.Fatal("no Redis limiter keys")
		}
		for _, key := range keys {
			if strings.Contains(key, h.victim) || strings.Contains(key, "10.0.0.1") || !strings.HasPrefix(key, "auth:abuse:login:") {
				t.Fatalf("unsafe/unexpected key %q", key)
			}
		}
	})

	t.Run("concurrent_pair_records_every_accepted_failure", func(t *testing.T) {
		h := newLoginIntegrationHarness(t, client, testLoginPolicy())
		const requests = 20
		start := make(chan struct{})
		results := make(chan int, requests)
		var wg sync.WaitGroup
		for range requests {
			wg.Add(1)
			go func() { defer wg.Done(); <-start; results <- h.login("10.0.0.2", h.victim, "wrong").Code }()
		}
		close(start)
		wg.Wait()
		close(results)
		accepted := 0
		for status := range results {
			if status == http.StatusUnauthorized {
				accepted++
			} else if status != http.StatusTooManyRequests {
				t.Fatalf("status=%d", status)
			}
		}
		count, _, err := h.limiter.Count(context.Background(), "login:ip-identifier", "10.0.0.2\x00"+h.victim)
		if err != nil || count < int64(accepted) || count < int64(h.policy.LoginIPIdentifierLimit) || count > requests {
			t.Fatalf("accepted=%d redis=%d err=%v", accepted, count, err)
		}
		if accepted > h.policy.LoginIPIdentifierLimit {
			t.Fatalf("accepted=%d exceeds pair allowance=%d", accepted, h.policy.LoginIPIdentifierLimit)
		}
		h.expectStatus(t, h.login("10.0.0.2", h.victim, "wrong"), http.StatusTooManyRequests)
		t.Logf("concurrent accepted failures=%d; Redis failure count=%d", accepted, count)
	})

	t.Run("different_ip_shared_nat_broad_and_reset", func(t *testing.T) {
		h := newLoginIntegrationHarness(t, client, testLoginPolicy())
		for i := 0; i < 5; i++ {
			h.expectStatus(t, h.login("10.0.0.3", h.victim, "wrong"), http.StatusUnauthorized)
		}
		h.expectStatus(t, h.login("10.0.0.4", h.victim, "correct-password"), http.StatusOK)
		policy := testLoginPolicy()
		policy.LoginBroadIPLimit = 4
		h = newLoginIntegrationHarness(t, client, policy)
		for i := 0; i < 4; i++ {
			h.expectStatus(t, h.login("10.0.0.5", fmt.Sprintf("user%d@example.test", i), "wrong"), http.StatusUnauthorized)
		}
		h.expectStatus(t, h.login("10.0.0.5", "user4@example.test", "wrong"), http.StatusTooManyRequests)

		h.reset(t)
		for i := 0; i < 4; i++ {
			h.expectStatus(t, h.login("10.0.0.6", fmt.Sprintf("user%d@example.test", i), "correct-password"), http.StatusOK)
		}
		count, _, err := h.limiter.Count(context.Background(), "login:ip", "10.0.0.6")
		if err != nil || count != 0 {
			t.Fatalf("successful NAT count=%d err=%v", count, err)
		}
		for i := 0; i < 2; i++ {
			h.expectStatus(t, h.login("10.0.0.7", h.victim, "wrong"), http.StatusUnauthorized)
		}
		h.expectStatus(t, h.login("10.0.0.7", h.victim, "correct-password"), http.StatusOK)
		pair, _, _ := h.limiter.Count(context.Background(), "login:ip-identifier", "10.0.0.7\x00"+h.victim)
		id, _, _ := h.limiter.Count(context.Background(), "login:identifier", h.victim)
		broad, _, _ := h.limiter.Count(context.Background(), "login:ip", "10.0.0.7")
		if pair != 0 || id != 0 || broad != 2 {
			t.Fatalf("pair=%d id=%d broad=%d", pair, id, broad)
		}
	})

	t.Run("soft_risk_unknown_and_missing_ip", func(t *testing.T) {
		policy := testLoginPolicy()
		policy.LoginIdentifierRiskLimit = 2
		policy.LoginIdentifierRiskDelay = 20 * time.Millisecond
		h := newLoginIntegrationHarness(t, client, policy)
		for i := 0; i < 2; i++ {
			h.expectStatus(t, h.login(fmt.Sprintf("10.0.1.%d", i), h.victim, "wrong"), http.StatusUnauthorized)
		}
		started := time.Now()
		h.expectStatus(t, h.login("10.0.1.9", h.victim, "correct-password"), http.StatusOK)
		if elapsed := time.Since(started); elapsed < 15*time.Millisecond || elapsed > time.Second {
			t.Fatalf("risk delay=%s", elapsed)
		}
		h.reset(t)
		known := h.login("10.0.2.1", h.victim, "wrong")
		unknown := h.login("10.0.2.2", "missing@example.test", "wrong")
		if known.Code != unknown.Code || bodyCode(t, known) != bodyCode(t, unknown) || bodyMessage(t, known) != bodyMessage(t, unknown) {
			t.Fatalf("known=%s unknown=%s", known.Body.String(), unknown.Body.String())
		}
		missing := h.loginWithHeaders("", h.victim, "correct-password", map[string]string{"X-Forwarded-For": "8.8.8.8", "X-Real-IP": "8.8.4.4"})
		h.expectStatus(t, missing, http.StatusServiceUnavailable)
	})
}

type loginIntegrationHarness struct {
	t       *testing.T
	client  *redis.Client
	limiter outbound.AuthAbuseLimiter
	engine  http.Handler
	policy  config.AuthAbuseConfig
	victim  string
}

func newLoginIntegrationHarness(t *testing.T, client *redis.Client, policy config.AuthAbuseConfig) *loginIntegrationHarness {
	t.Helper()
	h := &loginIntegrationHarness{t: t, client: client, policy: policy, victim: "victim@example.test"}
	h.reset(t)
	users := map[string]*entity.User{}
	for _, email := range append([]string{h.victim}, "user0@example.test", "user1@example.test", "user2@example.test", "user3@example.test", "user4@example.test") {
		users[email] = &entity.User{ID: email, Username: email, Email: email, Password: "correct-password", Role: rbac.RoleUser, IsActive: true}
	}
	h.limiter = redisadapter.NewAuthAbuseLimiter(client)
	uc := authusecase.NewLoginUseCaseWithAbuse(&integrationUsers{users: users}, integrationPassword{}, integrationJWT{}, integrationIAT{}, h.limiter, policy)
	r := NewRouter(&root.AuthHandler{Login: authhandler.NewLoginHandler(uc)}, nil, nil, func(c *gin.Context) { c.Next() }, zap.NewNop())
	r.engine.POST("/api/v1/auth/login", authhandler.NewLoginHandler(uc).Handle)
	h.engine = r.engine
	return h
}

func (h *loginIntegrationHarness) reset(t *testing.T) {
	t.Helper()
	if err := h.client.FlushDB(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
}
func (h *loginIntegrationHarness) login(ip, identifier, password string) *httptest.ResponseRecorder {
	return h.loginWithHeaders(ip, identifier, password, nil)
}
func (h *loginIntegrationHarness) loginWithHeaders(ip, identifier, password string, headers map[string]string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{"identifier": identifier, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if ip != "" {
		req.Header.Set("X-Client-IP", ip)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	out := httptest.NewRecorder()
	h.engine.ServeHTTP(out, req)
	return out
}
func (h *loginIntegrationHarness) expectStatus(t *testing.T, out *httptest.ResponseRecorder, want int) {
	t.Helper()
	if out.Code != want {
		t.Fatalf("status=%d want=%d body=%s", out.Code, want, out.Body.String())
	}
}
func bodyCode(t *testing.T, out *httptest.ResponseRecorder) int {
	t.Helper()
	var r response.APIResponse
	if err := json.Unmarshal(out.Body.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	return r.Code
}
func bodyMessage(t *testing.T, out *httptest.ResponseRecorder) string {
	t.Helper()
	var r response.APIResponse
	if err := json.Unmarshal(out.Body.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	return r.Msg
}

type integrationUsers struct{ users map[string]*entity.User }

func (r *integrationUsers) CreateUser(context.Context, *entity.User) error { return nil }
func (r *integrationUsers) GetUserByEmail(_ context.Context, email string) (*entity.User, error) {
	u, ok := r.users[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}
func (r *integrationUsers) GetUserByUsername(ctx context.Context, u string) (*entity.User, error) {
	return r.GetUserByEmail(ctx, u)
}
func (r *integrationUsers) GetUserById(context.Context, string) (*entity.User, error) {
	return nil, domain.ErrUserNotFound
}
func (r *integrationUsers) ListUsers(context.Context, outbound.ListUsersFilter) (outbound.ListUsersResult, error) {
	return outbound.ListUsersResult{}, nil
}
func (r *integrationUsers) SearchPublicUsers(context.Context, outbound.SearchPublicUsersFilter) (outbound.SearchPublicUsersResult, error) {
	return outbound.SearchPublicUsersResult{}, nil
}
func (r *integrationUsers) UpdateUser(context.Context, *entity.User) error { return nil }
func (r *integrationUsers) UpdatePassword(context.Context, string, string, time.Time) error {
	return nil
}
func (r *integrationUsers) UpdateProfile(context.Context, string, outbound.ProfileUpdates) error {
	return nil
}
func (r *integrationUsers) UpdateAvatar(context.Context, string, string, string, time.Time) error {
	return nil
}
func (r *integrationUsers) DeleteUser(context.Context, string) error { return nil }

type integrationPassword struct{}

func (integrationPassword) HashAndSalt([]byte) (string, error) { return "", nil }
func (integrationPassword) ComparePasswords(hash string, plain []byte) bool {
	return hash == string(plain)
}

type integrationJWT struct{}

func (integrationJWT) GenerateAccessToken(context.Context, string, string, rbac.Role) (string, int, error) {
	return "access", 60, nil
}
func (integrationJWT) GenerateRefreshToken(context.Context, string, string, rbac.Role) (string, int, error) {
	return "refresh", 60, nil
}
func (integrationJWT) VerifyAccessToken(context.Context, string) (string, string, rbac.Role, error) {
	return "", "", "", nil
}
func (integrationJWT) VerifyRefreshToken(context.Context, string) (string, string, rbac.Role, int64, error) {
	return "", "", "", 0, nil
}

type integrationIAT struct{}

func (integrationIAT) SetLogoutAllIAT(context.Context, string, int64) error   { return nil }
func (integrationIAT) GetLogoutAllIAT(context.Context, string) (int64, error) { return 0, nil }

var _ pkgauth.LogoutAllIATStore = integrationIAT{}

func testLoginPolicy() config.AuthAbuseConfig {
	return config.AuthAbuseConfig{LoginIPIdentifierLimit: 5, LoginIdentifierRiskLimit: 5, LoginIdentifierRiskDelay: 10 * time.Millisecond, LoginBroadIPLimit: 20, LoginWindow: 3 * time.Second}
}
