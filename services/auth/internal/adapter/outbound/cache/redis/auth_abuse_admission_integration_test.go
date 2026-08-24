package redis

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"go-judge-system/services/auth/internal/application/port/outbound"

	redisclient "github.com/redis/go-redis/v9"
)

// These tests execute admissionScript against a real, explicitly dedicated
// non-zero Redis DB. Example:
// AUTH_LOGIN_INTEGRATION_REDIS_URL=redis://127.0.0.1:6379/15 \
// go test ./internal/adapter/outbound/cache/redis -run TestAbuseAdmissionIntegration -count=1
func TestAbuseAdmissionIntegration(t *testing.T) {
	url := os.Getenv("AUTH_LOGIN_INTEGRATION_REDIS_URL")
	if url == "" {
		t.Skip("set AUTH_LOGIN_INTEGRATION_REDIS_URL to a dedicated Redis DB")
	}
	opts, err := redisclient.ParseURL(url)
	if err != nil {
		t.Fatal(err)
	}
	if opts.DB == 0 {
		t.Fatal("AUTH_LOGIN_INTEGRATION_REDIS_URL must select a dedicated non-zero Redis DB")
	}
	client := redisclient.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("real Redis ping: %v", err)
	}
	defer client.Close()

	t.Run("sequential_limit_ttl_retry_after_and_key_privacy", func(t *testing.T) {
		h := newAdmissionIntegrationHarness(t, client)
		req := admissionRequest("pair", "10.0.0.1\x00victim@example.test", 5, time.Minute)
		for range 5 {
			result := h.acquire(t, req)
			if !result.Allowed {
				t.Fatal("admission before boundary was denied")
			}
		}
		denied := h.acquire(t, req)
		if denied.Allowed || denied.RetryAfter <= 0 || denied.RetryAfter > time.Minute {
			t.Fatalf("denied=%+v", denied)
		}
		if count := h.count(t, "pair", "10.0.0.1\x00victim@example.test"); count != 5 {
			t.Fatalf("count=%d want 5", count)
		}
		key := authAbuseKey("pair", "10.0.0.1\x00victim@example.test")
		beforeDeniedTTL := h.ttl(t, key)
		if result := h.acquire(t, req); result.Allowed {
			t.Fatal("exhausted scope was unexpectedly admitted")
		}
		ttl := h.ttl(t, key)
		if ttl <= 0 || ttl > time.Minute {
			t.Fatalf("ttl=%s", ttl)
		}
		if ttl > beforeDeniedTTL {
			t.Fatalf("denial refreshed limiter TTL from %s to %s", beforeDeniedTTL, ttl)
		}
		for _, key := range h.keys(t) {
			if strings.Contains(key, "10.0.0.1") || strings.Contains(key, "victim@example.test") || !strings.HasPrefix(key, "auth:abuse:") {
				t.Fatalf("unsafe key %q", key)
			}
		}
	})

	t.Run("single_scope_concurrency_is_exact", func(t *testing.T) {
		h := newAdmissionIntegrationHarness(t, client)
		req := admissionRequest("pair", "10.0.0.2\x00victim@example.test", 5, time.Minute)
		if allowed := h.concurrentAcquire(t, req, 20); allowed != 5 {
			t.Fatalf("allowed=%d want 5", allowed)
		}
		if count := h.count(t, "pair", "10.0.0.2\x00victim@example.test"); count != 5 {
			t.Fatalf("count=%d want 5", count)
		}
	})

	t.Run("multi_scope_concurrency_commits_everything_or_nothing", func(t *testing.T) {
		h := newAdmissionIntegrationHarness(t, client)
		req := outbound.AdmissionRequest{Scopes: []outbound.AdmissionScope{
			{Purpose: "pair", Scope: "10.0.0.3\x00victim@example.test", Limit: 5, Window: time.Minute},
			{Purpose: "account-hour", Scope: "victim@example.test", Limit: 3, Window: time.Minute},
			{Purpose: "account-day", Scope: "victim@example.test", Limit: 10, Window: time.Minute},
			{Purpose: "broad-hour", Scope: "10.0.0.3", Limit: 100, Window: time.Minute},
		}}
		if allowed := h.concurrentAcquire(t, req, 20); allowed != 3 {
			t.Fatalf("allowed=%d want 3", allowed)
		}
		for _, scope := range req.Scopes {
			if count := h.count(t, scope.Purpose, scope.Scope); count != 3 {
				t.Fatalf("%s count=%d want 3", scope.Purpose, count)
			}
		}
	})

	t.Run("cooldown_concurrency_and_denial_do_not_mutate_counters", func(t *testing.T) {
		h := newAdmissionIntegrationHarness(t, client)
		req := outbound.AdmissionRequest{
			Scopes:   []outbound.AdmissionScope{{Purpose: "account-hour", Scope: "victim@example.test", Limit: 100, Window: time.Minute}},
			Cooldown: &outbound.CooldownScope{Purpose: "cooldown", Scope: "victim@example.test", Duration: time.Minute},
		}
		if allowed := h.concurrentAcquire(t, req, 20); allowed != 1 {
			t.Fatalf("allowed=%d want 1", allowed)
		}
		if count := h.count(t, "account-hour", "victim@example.test"); count != 1 {
			t.Fatalf("count=%d want 1", count)
		}
		cooldownTTL := h.ttl(t, authAbuseKey("cooldown", "victim@example.test"))
		if cooldownTTL <= 0 || cooldownTTL > time.Minute {
			t.Fatalf("cooldown ttl=%s", cooldownTTL)
		}
	})

	t.Run("exhausted_scope_never_mutates_available_scope", func(t *testing.T) {
		h := newAdmissionIntegrationHarness(t, client)
		a := admissionRequest("scope-a", "victim@example.test", 2, time.Minute)
		b := admissionRequest("scope-b", "10.0.0.4", 10, time.Minute)
		for range 2 {
			if result := h.acquire(t, a); !result.Allowed {
				t.Fatal("could not exhaust scope A")
			}
		}
		for range 2 {
			if result := h.acquire(t, b); !result.Allowed {
				t.Fatal("could not seed scope B")
			}
		}
		result := h.acquire(t, outbound.AdmissionRequest{Scopes: []outbound.AdmissionScope{
			{Purpose: "scope-a", Scope: "victim@example.test", Limit: 2, Window: time.Minute},
			{Purpose: "scope-b", Scope: "10.0.0.4", Limit: 10, Window: time.Minute},
		}})
		if result.Allowed || result.RetryAfter <= 0 {
			t.Fatalf("result=%+v", result)
		}
		if count := h.count(t, "scope-b", "10.0.0.4"); count != 2 {
			t.Fatalf("available scope B mutated to %d", count)
		}
	})

	t.Run("retry_after_uses_the_longest_blocking_constraint", func(t *testing.T) {
		h := newAdmissionIntegrationHarness(t, client)
		short := admissionRequest("short", "victim@example.test", 1, time.Minute)
		long := admissionRequest("long", "victim@example.test", 1, 2*time.Minute)
		if !h.acquire(t, short).Allowed || !h.acquire(t, long).Allowed {
			t.Fatal("could not seed blocking scopes")
		}
		result := h.acquire(t, outbound.AdmissionRequest{Scopes: []outbound.AdmissionScope{
			{Purpose: "short", Scope: "victim@example.test", Limit: 1, Window: time.Minute},
			{Purpose: "long", Scope: "victim@example.test", Limit: 1, Window: 2 * time.Minute},
		}})
		if result.Allowed || result.RetryAfter <= time.Minute {
			t.Fatalf("result=%+v, want retry after from long window", result)
		}
	})
}

type admissionIntegrationHarness struct {
	t      *testing.T
	client *redisclient.Client
	store  outbound.AbuseAdmission
}

func newAdmissionIntegrationHarness(t *testing.T, client *redisclient.Client) *admissionIntegrationHarness {
	t.Helper()
	if err := client.FlushDB(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	return &admissionIntegrationHarness{t: t, client: client, store: NewAbuseAdmission(client)}
}

func admissionRequest(purpose, scope string, limit int, window time.Duration) outbound.AdmissionRequest {
	return outbound.AdmissionRequest{Scopes: []outbound.AdmissionScope{{Purpose: purpose, Scope: scope, Limit: limit, Window: window}}}
}

func (h *admissionIntegrationHarness) acquire(t *testing.T, request outbound.AdmissionRequest) outbound.AdmissionResult {
	t.Helper()
	result, err := h.store.Acquire(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func (h *admissionIntegrationHarness) concurrentAcquire(t *testing.T, request outbound.AdmissionRequest, total int) int {
	t.Helper()
	start := make(chan struct{})
	results := make(chan outbound.AdmissionResult, total)
	errs := make(chan error, total)
	var wg sync.WaitGroup
	for range total {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result, err := h.store.Acquire(context.Background(), request)
			if err != nil {
				errs <- err
				return
			}
			results <- result
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	allowed := 0
	for result := range results {
		if result.Allowed {
			allowed++
		} else if result.RetryAfter <= 0 {
			t.Errorf("denied result missing retry-after: %+v", result)
		}
	}
	return allowed
}

func (h *admissionIntegrationHarness) count(t *testing.T, purpose, scope string) int64 {
	t.Helper()
	count, err := h.client.Get(context.Background(), authAbuseKey(purpose, scope)).Int64()
	if err != nil {
		t.Fatal(err)
	}
	return count
}

func (h *admissionIntegrationHarness) ttl(t *testing.T, key string) time.Duration {
	t.Helper()
	ttl, err := h.client.PTTL(context.Background(), key).Result()
	if err != nil {
		t.Fatal(err)
	}
	return ttl
}

func (h *admissionIntegrationHarness) keys(t *testing.T) []string {
	t.Helper()
	keys, err := h.client.Keys(context.Background(), "auth:abuse:*").Result()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) == 0 {
		t.Fatal("no admission keys")
	}
	return keys
}
