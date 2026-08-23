package cooldown

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"go-judge-system/pkg/config"

	"github.com/redis/go-redis/v9"
)

func TestSubmissionCooldownKeyHashesScope(t *testing.T) {
	key := submissionCooldownKey("user-a@example.test", 42)
	if !strings.HasPrefix(key, submissionCooldownKeyPrefix) {
		t.Fatalf("key = %q", key)
	}
	if strings.Contains(key, "user-a@example.test") || strings.Contains(key, "\x00") {
		t.Fatalf("key leaks raw scope: %q", key)
	}
	if len(strings.TrimPrefix(key, submissionCooldownKeyPrefix)) != 64 {
		t.Fatalf("key hash length = %d, want 64", len(strings.TrimPrefix(key, submissionCooldownKeyPrefix)))
	}
}

func TestNewRedisSubmissionCooldownRejectsInvalidConfiguration(t *testing.T) {
	if _, err := NewRedisSubmissionCooldown(nil, config.SubmissionConfig{SubmitCooldown: time.Second}); err == nil {
		t.Fatal("nil Redis client was accepted")
	}
	if _, err := NewRedisSubmissionCooldown(redis.NewClient(&redis.Options{}), config.SubmissionConfig{}); err == nil {
		t.Fatal("zero cooldown was accepted")
	}
}

func TestRedisSubmissionCooldownFailureNeverAllows(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 20 * time.Millisecond, ReadTimeout: 20 * time.Millisecond, WriteTimeout: 20 * time.Millisecond})
	defer client.Close()

	cooldown, err := NewRedisSubmissionCooldown(client, config.SubmissionConfig{SubmitCooldown: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, err := cooldown.Acquire(ctx, "user-a", 42)
	if err == nil || result.Allowed {
		t.Fatalf("result/error = %+v/%v, want denied error", result, err)
	}
}

// These tests use a dedicated non-zero Redis DB and never flush it. Example:
// SUBMISSION_COOLDOWN_INTEGRATION_REDIS_URL=redis://127.0.0.1:6379/14 \
// go test ./internal/adapter/outbound/cooldown -run TestRedisSubmissionCooldownIntegration -count=1
func TestRedisSubmissionCooldownIntegration(t *testing.T) {
	url := os.Getenv("SUBMISSION_COOLDOWN_INTEGRATION_REDIS_URL")
	if url == "" {
		t.Skip("set SUBMISSION_COOLDOWN_INTEGRATION_REDIS_URL to a dedicated Redis DB")
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		t.Fatal(err)
	}
	if opts.DB == 0 {
		t.Fatal("SUBMISSION_COOLDOWN_INTEGRATION_REDIS_URL must select a dedicated non-zero Redis DB")
	}
	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("real Redis ping: %v", err)
	}
	defer client.Close()

	newCooldown := func(t *testing.T, duration time.Duration) (*redisSubmissionCooldown, string) {
		t.Helper()
		service, err := NewRedisSubmissionCooldown(client, config.SubmissionConfig{SubmitCooldown: duration})
		if err != nil {
			t.Fatal(err)
		}
		cooldown, ok := service.(*redisSubmissionCooldown)
		if !ok {
			t.Fatalf("cooldown type = %T", service)
		}
		userID := fmt.Sprintf("integration-user-%d", time.Now().UnixNano())
		return cooldown, userID
	}

	t.Run("concurrent_acquire_is_exact_and_expires", func(t *testing.T) {
		const attempts = 20
		cooldown, userID := newCooldown(t, 150*time.Millisecond)
		key := submissionCooldownKey(userID, 42)
		t.Cleanup(func() { _ = client.Del(context.Background(), key).Err() })

		start := make(chan struct{})
		results := make(chan bool, attempts)
		var wg sync.WaitGroup
		for range attempts {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				result, err := cooldown.Acquire(context.Background(), userID, 42)
				if err != nil {
					t.Errorf("Acquire() error = %v", err)
					return
				}
				if !result.Allowed && result.RetryAfter <= 0 {
					t.Errorf("denied result missing retry-after: %+v", result)
				}
				results <- result.Allowed
			}()
		}
		close(start)
		wg.Wait()
		close(results)

		allowed := 0
		for result := range results {
			if result {
				allowed++
			}
		}
		if allowed != 1 {
			t.Fatalf("allowed=%d, want 1", allowed)
		}
		ttl, err := client.PTTL(context.Background(), key).Result()
		if err != nil || ttl <= 0 || ttl > 150*time.Millisecond {
			t.Fatalf("ttl/error = %s/%v", ttl, err)
		}
		if strings.Contains(key, userID) || strings.Contains(key, "\x00") {
			t.Fatalf("key leaks raw scope: %q", key)
		}

		time.Sleep(225 * time.Millisecond)
		result, err := cooldown.Acquire(context.Background(), userID, 42)
		if err != nil || !result.Allowed {
			t.Fatalf("acquire after expiry = %+v/%v", result, err)
		}
	})

	t.Run("different_user_or_problem_is_independent", func(t *testing.T) {
		cooldown, userID := newCooldown(t, time.Second)
		keys := []string{
			submissionCooldownKey(userID, 42),
			submissionCooldownKey(userID, 43),
			submissionCooldownKey(userID+"-other", 42),
		}
		t.Cleanup(func() { _ = client.Del(context.Background(), keys...).Err() })

		for _, scope := range []struct {
			userID    string
			problemID int64
		}{
			{userID, 42},
			{userID, 43},
			{userID + "-other", 42},
		} {
			result, err := cooldown.Acquire(context.Background(), scope.userID, scope.problemID)
			if err != nil || !result.Allowed {
				t.Fatalf("Acquire(%q, %d) = %+v/%v", scope.userID, scope.problemID, result, err)
			}
		}
	})
}
