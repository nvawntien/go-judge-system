package requestctx

import (
	"context"
	"testing"
)

func TestClientIP(t *testing.T) {
	base := context.WithValue(context.Background(), struct{ key string }{"request-id"}, "request-1")
	ctx := WithClientIP(base, "203.0.113.10")
	if got, ok := ClientIP(ctx); !ok || got != "203.0.113.10" {
		t.Fatalf("ClientIP() = %q, %v", got, ok)
	}
	if got := ctx.Value(struct{ key string }{"request-id"}); got != "request-1" {
		t.Fatalf("unrelated context value = %v", got)
	}
}

func TestClientIPMissing(t *testing.T) {
	if got, ok := ClientIP(context.Background()); ok || got != "" {
		t.Fatalf("ClientIP() = %q, %v", got, ok)
	}
}
