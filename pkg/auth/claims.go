package auth

import (
	"go-judge-system/pkg/rbac"

	"github.com/gin-gonic/gin"
)

type Claims struct {
	UserID        string
	Username      string
	Role          rbac.Role
	TokenIssuedAt int64
}

const claimsContextKey = "auth_claims"

func SetClaims(c *gin.Context, claims Claims) {
	c.Set(claimsContextKey, claims)
}

func GetClaims(c *gin.Context) (Claims, bool) {
	val, ok := c.Get(claimsContextKey)
	if !ok {
		return Claims{}, false
	}

	claims, ok := val.(Claims)
	return claims, ok
}