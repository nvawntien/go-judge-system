package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-judge-system/pkg/requestctx"

	"github.com/gin-gonic/gin"
)

func TestRequestMetadataMiddlewareUsesTrustedHeaderOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(NewRequestMetadataMiddleware())
	r.GET("/", func(c *gin.Context) {
		ip, ok := requestctx.ClientIP(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"ip": ip, "ok": ok})
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Client-IP", " 203.0.113.8 ")
	request.Header.Set("X-Forwarded-For", "198.51.100.1")
	request.Header.Set("X-Real-IP", "198.51.100.2")
	response := httptest.NewRecorder()
	r.ServeHTTP(response, request)
	if got := response.Body.String(); got != `{"ip":"203.0.113.8","ok":true}` {
		t.Fatalf("body=%s", got)
	}
}

func TestRequestMetadataMiddlewareDoesNotUseForwardingFallbacks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(NewRequestMetadataMiddleware())
	r.GET("/", func(c *gin.Context) {
		_, ok := requestctx.ClientIP(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"ok": ok})
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Forwarded-For", "198.51.100.1")
	request.Header.Set("X-Real-IP", "198.51.100.2")
	response := httptest.NewRecorder()
	r.ServeHTTP(response, request)
	if got := response.Body.String(); got != `{"ok":false}` {
		t.Fatalf("body=%s", got)
	}
}
