package auth

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

// clientIP accepts only Envoy's overwritten X-Client-IP. The fallback is the
// immediate peer for internal/direct service calls; it never parses XFF/X-Real-IP.
func clientIP(c *gin.Context) string {
	if ip := net.ParseIP(strings.TrimSpace(c.GetHeader("X-Client-IP"))); ip != nil {
		return ip.String()
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err == nil && net.ParseIP(host) != nil {
		return host
	}
	return "unknown"
}
