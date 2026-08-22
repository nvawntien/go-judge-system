package auth

import (
	"context"
	"math"
	"strings"
	"time"

	"go-judge-system/pkg/config"
	"go-judge-system/pkg/requestctx"
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/port/outbound"
)

type authAbuse struct {
	limiter outbound.AuthAbuseLimiter
	policy  config.AuthAbuseConfig
}

func (a authAbuse) clientIP(ctx context.Context) (string, error) {
	ip, ok := requestctx.ClientIP(ctx)
	if !ok {
		return "", response.NewAppError(response.CodeServiceUnavailable, "authentication protection temporarily unavailable", nil)
	}
	return ip, nil
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

func (a authAbuse) checkFailureLimit(ctx context.Context, purpose, scope string, limit int) error {
	count, retry, err := a.limiter.Count(ctx, purpose, scope)
	if err != nil {
		return response.NewAppError(response.CodeServiceUnavailable, "authentication protection temporarily unavailable", err)
	}
	if count >= int64(limit) {
		return response.NewRateLimitError("too many requests, please try again later", retry)
	}
	return nil
}

// recordFailure uses the existing atomic fixed-window Lua increment without
// imposing a threshold here; hard decisions are made only by pre-checks.
func (a authAbuse) recordFailure(ctx context.Context, purpose, scope string, window time.Duration) error {
	_, _, err := a.limiter.Allow(ctx, purpose, scope, math.MaxInt, window)
	if err != nil {
		return response.NewAppError(response.CodeServiceUnavailable, "authentication protection temporarily unavailable", err)
	}
	return nil
}

func (a authAbuse) delayForIdentifierRisk(ctx context.Context, identifier string) error {
	count, _, err := a.limiter.Count(ctx, "login:identifier", identifier)
	if err != nil {
		return response.NewAppError(response.CodeServiceUnavailable, "authentication protection temporarily unavailable", err)
	}
	if count < int64(a.policy.LoginIdentifierRiskLimit) || a.policy.LoginIdentifierRiskDelay <= 0 {
		return nil
	}
	timer := time.NewTimer(a.policy.LoginIdentifierRiskDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func loginIPIdentifierScope(clientIP, identifier string) string {
	return clientIP + "\x00" + identifier
}
func normalizedIdentifier(value string) string { return strings.ToLower(strings.TrimSpace(value)) }
