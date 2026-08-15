package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-judge-system/pkg/auth"
	pkgmiddleware "go-judge-system/pkg/middleware"
	"go-judge-system/pkg/rbac"

	"github.com/gin-gonic/gin"
)

func TestModeratorProblemRoutesRoleMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)

	operations := []struct {
		name   string
		method string
		path   string
	}{
		{name: "catalog list", method: http.MethodGet, path: "/api/v1/admin/problems"},
		{name: "publish", method: http.MethodPatch, path: "/api/v1/admin/problems/42/publish"},
	}
	for _, operation := range operations {
		operation := operation
		t.Run(operation.name, func(t *testing.T) {
			for _, role := range []rbac.Role{rbac.RoleUser, rbac.RoleContributor, rbac.RoleModerator, rbac.RoleAdmin} {
				role := role
				t.Run(string(role), func(t *testing.T) {
					called := false
					engine := gin.New()
					engine.Handle(
						operation.method,
						operation.path,
						func(c *gin.Context) {
							auth.SetClaims(c, auth.Claims{UserID: "actor", Role: role})
							c.Next()
						},
						pkgmiddleware.RequireRole(rbac.RoleModerator),
						func(c *gin.Context) {
							called = true
							c.Status(http.StatusOK)
						},
					)

					request := httptest.NewRequest(operation.method, operation.path, nil)
					response := httptest.NewRecorder()
					engine.ServeHTTP(response, request)

					allowed := role.AtLeast(rbac.RoleModerator)
					if allowed && (!called || response.Code != http.StatusOK) {
						t.Fatalf("called/status = %v/%d, want true/200", called, response.Code)
					}
					if !allowed && (called || response.Code != http.StatusForbidden) {
						t.Fatalf("called/status = %v/%d, want false/403", called, response.Code)
					}
				})
			}
		})
	}
}
