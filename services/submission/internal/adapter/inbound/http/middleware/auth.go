package middleware

import (
	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/response"

	"github.com/gin-gonic/gin"
)

// NewAuthMiddleware extracts user identity from X-User-* headers set by KrakenD gateway.
func NewAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			response.Error(c, response.CodeUnauthorized, "unauthorized: missing identity headers")
			return
		}

		auth.SetClaims(c, auth.Claims{
			UserID:   userID,
			Username: c.GetHeader("X-Username"),
			Role:     c.GetHeader("X-Role"),
		})

		c.Next()
	}
}
