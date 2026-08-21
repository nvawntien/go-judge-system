// Package requestctx carries transport-derived metadata through standard Go
// contexts without coupling application code to a specific HTTP framework.
package requestctx

import "context"

type clientIPKey struct{}

func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey{}, ip)
}

func ClientIP(ctx context.Context) (string, bool) {
	ip, ok := ctx.Value(clientIPKey{}).(string)
	return ip, ok && ip != ""
}
