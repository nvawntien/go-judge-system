package middleware

import (
	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/pkg/response"

	"github.com/gin-gonic/gin"
)

func RequireRole(min rbac.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := auth.GetClaims(c)
		if !ok {
			response.Error(c, response.CodeUnauthorized, "unauthorized")
			return
		}

		if !claims.Role.AtLeast(min) {
			response.Error(c, response.CodeForbidden, "forbidden: insufficient role")
			return
		}

		c.Next()
	}
}
