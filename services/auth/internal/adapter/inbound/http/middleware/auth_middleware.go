package middleware

import (
	"go-judge-system/services/auth/internal/adapter/inbound/http/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

// NewAuthMiddleware creates a middleware that trusts the KrakenD API Gateway.
// It extracts the user identity from X-User-* headers set by the gateway's JWT validator.
func NewAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			response.Error(c, http.StatusUnauthorized, "unauthorized: missing identity headers from gateway")
			return
		}

		c.Set("user_id", userID)
		c.Set("username", c.GetHeader("X-Username"))
		c.Set("role", c.GetHeader("X-Role"))

		c.Next()
	}
}
