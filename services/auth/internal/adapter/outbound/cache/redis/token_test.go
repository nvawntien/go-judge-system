package redis

import (
	"testing"

	"go-judge-system/services/auth/internal/application/port/outbound"
)

func TestTokenKeysArePurposeScoped(t *testing.T) {
	const (
		identifier = "user-1"
		hash       = "hashed-token"
	)

	verifyToken := tokenKey(outbound.TokenPurposeVerifyEmail, hash)
	resetToken := tokenKey(outbound.TokenPurposeResetPassword, hash)
	verifyLatest := latestTokenKey(outbound.TokenPurposeVerifyEmail, identifier)
	resetLatest := latestTokenKey(outbound.TokenPurposeResetPassword, identifier)
	verifyCooldown := resendCooldownKey(outbound.TokenPurposeVerifyEmail, identifier)
	resetCooldown := resendCooldownKey(outbound.TokenPurposeResetPassword, identifier)

	for name, pair := range map[string]struct{ got, other string }{
		"token":    {verifyToken, resetToken},
		"latest":   {verifyLatest, resetLatest},
		"cooldown": {verifyCooldown, resetCooldown},
	} {
		if pair.got == pair.other {
			t.Fatalf("%s keys collide: %q", name, pair.got)
		}
	}
}

func TestAuthAbuseKeysHashSensitiveScope(t *testing.T) {
	key := authAbuseKey("login:identifier", "person@example.test")
	if key == "" || key == "auth:abuse:login:identifier:person@example.test" {
		t.Fatalf("key exposes or omits scope hash: %q", key)
	}
}
