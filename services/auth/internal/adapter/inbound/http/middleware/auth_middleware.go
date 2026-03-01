package middleware

import (
	"go-judge-system/services/auth/internal/adapter/inbound/http/response"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func NewAuthMiddleware(jwtProvider outbound.JWTProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("access_token")
		if err != nil || token == "" {
			// Fallback: check Authorization header
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				response.Error(c, http.StatusUnauthorized, "unauthorized: missing access token")
				return
			}
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}

		userID, username, role, err := jwtProvider.VerifyAccessToken(c.Request.Context(), token)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "unauthorized: invalid or expired token")
			return
		}

		c.Set("user_id", userID)
		c.Set("username", username)
		c.Set("role", role)
		c.Next()
	}
}
