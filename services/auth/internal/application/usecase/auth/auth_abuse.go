package auth

import (
	"context"
	"strings"
	"time"

	"go-judge-system/pkg/config"
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/port/outbound"
)

type authAbuse struct {
	limiter outbound.AuthAbuseLimiter
	policy  config.AuthAbuseConfig
}

func (a authAbuse) allow(ctx context.Context, purpose, scope string, limit int, window time.Duration) (time.Duration, error) {
	ok, retry, err := a.limiter.Allow(ctx, purpose, scope, limit, window)
	if err != nil {
		return 0, response.NewAppError(response.CodeServiceUnavailable, "authentication protection temporarily unavailable", err)
	}
	if !ok {
		return retry, response.NewRateLimitError("too many requests, please try again later", retry)
	}
	return 0, nil
}
func normalizedIdentifier(value string) string { return strings.ToLower(strings.TrimSpace(value)) }
