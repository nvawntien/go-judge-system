package redis

import (
	"context"
	"strings"
	"testing"
	"time"

	"go-judge-system/services/auth/internal/application/port/outbound"

	redisclient "github.com/redis/go-redis/v9"
)

func TestAdmissionArgumentsValidateDefinitionsAndHashScopes(t *testing.T) {
	tests := []struct {
		name string
		req  outbound.AdmissionRequest
	}{
		{name: "empty request", req: outbound.AdmissionRequest{}},
		{name: "empty purpose", req: outbound.AdmissionRequest{Scopes: []outbound.AdmissionScope{{Scope: "scope", Limit: 1, Window: time.Minute}}}},
		{name: "empty scope", req: outbound.AdmissionRequest{Scopes: []outbound.AdmissionScope{{Purpose: "purpose", Limit: 1, Window: time.Minute}}}},
		{name: "zero limit", req: outbound.AdmissionRequest{Scopes: []outbound.AdmissionScope{{Purpose: "purpose", Scope: "scope", Window: time.Minute}}}},
		{name: "zero window", req: outbound.AdmissionRequest{Scopes: []outbound.AdmissionScope{{Purpose: "purpose", Scope: "scope", Limit: 1}}}},
		{name: "sub millisecond window", req: outbound.AdmissionRequest{Scopes: []outbound.AdmissionScope{{Purpose: "purpose", Scope: "scope", Limit: 1, Window: time.Nanosecond}}}},
		{name: "zero cooldown", req: outbound.AdmissionRequest{Cooldown: &outbound.CooldownScope{Purpose: "cooldown", Scope: "scope"}}},
		{name: "sub millisecond cooldown", req: outbound.AdmissionRequest{Cooldown: &outbound.CooldownScope{Purpose: "cooldown", Scope: "scope", Duration: time.Nanosecond}}},
		{name: "duplicate key", req: outbound.AdmissionRequest{Scopes: []outbound.AdmissionScope{
			{Purpose: "purpose", Scope: "scope", Limit: 1, Window: time.Minute},
			{Purpose: "purpose", Scope: "scope", Limit: 2, Window: time.Minute},
		}}},
		{name: "cooldown duplicates window", req: outbound.AdmissionRequest{
			Scopes:   []outbound.AdmissionScope{{Purpose: "purpose", Scope: "scope", Limit: 1, Window: time.Minute}},
			Cooldown: &outbound.CooldownScope{Purpose: "purpose", Scope: "scope", Duration: time.Minute},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := admissionArguments(tt.req); err == nil {
				t.Fatal("expected invalid admission definition error")
			}
		})
	}

	keys, _, err := admissionArguments(outbound.AdmissionRequest{
		Scopes: []outbound.AdmissionScope{{
			Purpose: "resend:ip-account", Scope: "203.0.113.9\x00person@example.test", Limit: 5, Window: time.Minute,
		}},
		Cooldown: &outbound.CooldownScope{Purpose: "resend:cooldown", Scope: "person@example.test", Duration: time.Minute},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		if !strings.HasPrefix(key, "auth:abuse:") || strings.Contains(key, "203.0.113.9") || strings.Contains(key, "person@example.test") {
			t.Fatalf("unsafe admission key %q", key)
		}
	}
	if authAbuseKey("resend:account-hour", "person@example.test") == authAbuseKey("forgot:account-hour", "person@example.test") {
		t.Fatal("different admission purposes must not share a counter")
	}
}

func TestAbuseAdmissionRedisFailureNeverAllows(t *testing.T) {
	client := redisclient.NewClient(&redisclient.Options{Addr: "127.0.0.1:0", DialTimeout: 10 * time.Millisecond, ReadTimeout: 10 * time.Millisecond, WriteTimeout: 10 * time.Millisecond})
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	result, err := NewAbuseAdmission(client).Acquire(ctx, outbound.AdmissionRequest{Scopes: []outbound.AdmissionScope{{
		Purpose: "test", Scope: "scope", Limit: 1, Window: time.Minute,
	}}})
	if err == nil {
		t.Fatal("expected Redis admission error")
	}
	if result.Allowed {
		t.Fatal("Redis failure must never allow admission")
	}
}
