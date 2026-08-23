package cooldown

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go-judge-system/pkg/config"
	"go-judge-system/services/submission/internal/application/port/outbound"

	"github.com/redis/go-redis/v9"
)

const submissionCooldownKeyPrefix = "submission:cooldown:"

type redisSubmissionCooldown struct {
	rdb      *redis.Client
	duration time.Duration
}

func NewRedisSubmissionCooldown(rdb *redis.Client, cfg config.SubmissionConfig) (outbound.SubmissionCooldown, error) {
	if rdb == nil {
		return nil, fmt.Errorf("redis client is required")
	}
	if cfg.SubmitCooldown <= 0 {
		return nil, fmt.Errorf("submission submit_cooldown must be greater than zero")
	}

	return &redisSubmissionCooldown{rdb: rdb, duration: cfg.SubmitCooldown}, nil
}

func (s *redisSubmissionCooldown) Acquire(ctx context.Context, userID string, problemID int64) (outbound.SubmissionCooldownResult, error) {
	if strings.TrimSpace(userID) == "" {
		return outbound.SubmissionCooldownResult{}, fmt.Errorf("user ID is required")
	}
	if problemID <= 0 {
		return outbound.SubmissionCooldownResult{}, fmt.Errorf("problem ID must be greater than zero")
	}

	key := submissionCooldownKey(userID, problemID)
	allowed, err := s.rdb.SetNX(ctx, key, "1", s.duration).Result()
	if err != nil {
		return outbound.SubmissionCooldownResult{}, fmt.Errorf("acquire submission cooldown: %w", err)
	}
	if allowed {
		return outbound.SubmissionCooldownResult{Allowed: true}, nil
	}

	retryAfter, err := s.rdb.PTTL(ctx, key).Result()
	if err != nil {
		return outbound.SubmissionCooldownResult{}, fmt.Errorf("read submission cooldown ttl: %w", err)
	}
	if retryAfter < 0 {
		return outbound.SubmissionCooldownResult{}, fmt.Errorf("submission cooldown key has no expiry")
	}
	if retryAfter == 0 {
		retryAfter = time.Millisecond
	}

	return outbound.SubmissionCooldownResult{RetryAfter: retryAfter}, nil
}

func submissionCooldownKey(userID string, problemID int64) string {
	scope := userID + "\x00" + strconv.FormatInt(problemID, 10)
	hash := sha256.Sum256([]byte(scope))
	return submissionCooldownKeyPrefix + hex.EncodeToString(hash[:])
}
