package auth

import "github.com/gin-gonic/gin"

type Claims struct {
	UserID   string
	Username string
	Role     string
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
