package middleware

import (
	"strconv"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/pkg/response"

	"github.com/gin-gonic/gin"
)

// NewAuthMiddleware creates a middleware that trusts the KrakenD API Gateway.
// It extracts the user identity from X-User-* headers set by the gateway's JWT validator.
func NewAuthMiddleware(logoutAllStore auth.LogoutAllIATStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			response.Error(c, response.CodeUnauthorized, "unauthorized: missing identity headers from gateway")
			return
		}

		tokenIAT, ok := parseTokenIAT(c.GetHeader("X-Token-Iat"))
		if !ok {
			response.Error(c, response.CodeUnauthorized, "unauthorized: missing or invalid token iat")
			return
		}

		logoutAllIAT, err := logoutAllStore.GetLogoutAllIAT(c.Request.Context(), userID)
		if err != nil {
			response.HandleError(c, response.NewAppError(response.CodeRedisError, "failed to validate session", err))
			return
		}

		if logoutAllIAT > 0 && tokenIAT <= logoutAllIAT {
			response.Error(c, response.CodeUnauthorized, "unauthorized: token has been invalidated")
			return
		}

		auth.SetClaims(c, auth.Claims{
			UserID:        userID,
			Username:      c.GetHeader("X-Username"),
			Role:          rbac.Role(c.GetHeader("X-Role")),
			TokenIssuedAt: tokenIAT,
		})

		c.Next()
	}
}

func parseTokenIAT(raw string) (int64, bool) {
	if raw == "" {
		return 0, false
	}

	iat, err := strconv.ParseInt(raw, 10, 64)
	if err == nil {
		return iat, true
	}

	floatIAT, floatErr := strconv.ParseFloat(raw, 64)
	if floatErr != nil {
		return 0, false
	}

	return int64(floatIAT), true
}
