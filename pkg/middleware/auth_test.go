package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-judge-system/pkg/auth"

	"github.com/gin-gonic/gin"
)

func TestAuthMiddlewareRejectsTokenAtOrBeforeInvalidationCutoff(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NewAuthMiddleware(middlewareIATStore{cutoff: 100}))
	router.GET("/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-User-ID", "user-1")
	req.Header.Set("X-Token-Iat", "100")
	req.Header.Set("X-Role", "user")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

type middlewareIATStore struct{ cutoff int64 }

func (s middlewareIATStore) SetLogoutAllIAT(context.Context, string, int64) error { return nil }
func (s middlewareIATStore) GetLogoutAllIAT(context.Context, string) (int64, error) {
	return s.cutoff, nil
}

var _ auth.LogoutAllIATStore = middlewareIATStore{}
