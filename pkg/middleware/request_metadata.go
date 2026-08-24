package middleware

import (
	"net"
	"strings"

	"go-judge-system/pkg/requestctx"

	"github.com/gin-gonic/gin"
)

// NewRequestMetadataMiddleware accepts only the trusted, edge-overwritten
// X-Client-IP header. It deliberately does not inspect X-Forwarded-For,
// X-Real-IP, or the direct peer address.
func NewRequestMetadataMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if ip := net.ParseIP(strings.TrimSpace(c.GetHeader("X-Client-IP"))); ip != nil {
			c.Request = c.Request.WithContext(requestctx.WithClientIP(c.Request.Context(), ip.String()))
		}
		c.Next()
	}
}
