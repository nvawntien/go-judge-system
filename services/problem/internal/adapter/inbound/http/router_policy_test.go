package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-judge-system/pkg/auth"
	pkgmiddleware "go-judge-system/pkg/middleware"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/problem/internal/adapter/inbound/http/handler/user/tag"
	"go-judge-system/services/problem/internal/application/dto"
	userinbound "go-judge-system/services/problem/internal/application/port/inbound/user"

	"github.com/gin-gonic/gin"
)

type publicTagCatalogUseCase struct{}

var _ userinbound.ListTagsUseCase = publicTagCatalogUseCase{}

func (publicTagCatalogUseCase) Execute(context.Context) (dto.ListTagsResponse, error) {
	return dto.ListTagsResponse{Items: []dto.TagResponse{{ID: 1, Name: "Graphs", Slug: "graphs"}}}, nil
}

func TestPublicTagCatalogDoesNotRequireAuthClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/api/v1/tags", tag.NewListTagsHandler(publicTagCatalogUseCase{}).Handle)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
}

func TestModeratorAdminRoutesRoleMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)

	operations := []struct {
		name   string
		method string
		path   string
	}{
		{name: "catalog list", method: http.MethodGet, path: "/api/v1/admin/problems"},
		{name: "publish", method: http.MethodPatch, path: "/api/v1/admin/problems/42/publish"},
		{name: "tag catalog administration", method: http.MethodGet, path: "/api/v1/admin/tags"},
		{name: "create tag", method: http.MethodPost, path: "/api/v1/admin/tags"},
		{name: "update tag", method: http.MethodPut, path: "/api/v1/admin/tags/5"},
		{name: "delete tag", method: http.MethodDelete, path: "/api/v1/admin/tags/5"},
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
