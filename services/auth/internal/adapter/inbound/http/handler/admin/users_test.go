package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/middleware"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

func TestAdminUsersHandlerBindsRoutesAndDoesNotExposeSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := &fakeAdminUsersUseCase{}
	handler := NewAdminUsersHandler(uc)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		auth.SetClaims(c, auth.Claims{UserID: "admin", Role: rbac.RoleAdmin})
		c.Next()
	})
	admin := router.Group("/api/v1/admin", middleware.RequireRole(rbac.RoleAdmin))
	admin.GET("/users", handler.List)
	admin.GET("/users/:user_id", handler.Get)
	admin.PATCH("/users/:user_id/suspension", handler.SetSuspension)

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?page=2&limit=10&search=ada&role=admin&status=active", nil))
	if list.Code != http.StatusOK || uc.listReq.Page == nil || *uc.listReq.Page != 2 || uc.listReq.Limit == nil || *uc.listReq.Limit != 10 || uc.listReq.Search != "ada" || uc.listReq.Role != "admin" || uc.listReq.Status != "active" {
		t.Fatalf("list status=%d request=%+v", list.Code, uc.listReq)
	}
	if body := list.Body.String(); containsAny(body, "password", "avatar_object_key") {
		t.Fatalf("response leaked sensitive field: %s", body)
	}

	detail := httptest.NewRecorder()
	router.ServeHTTP(detail, httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/5a35d699-5d11-4a5b-93ea-aa868cb1b75b", nil))
	if detail.Code != http.StatusOK || uc.userID != "5a35d699-5d11-4a5b-93ea-aa868cb1b75b" {
		t.Fatalf("detail status=%d userID=%q", detail.Code, uc.userID)
	}

	patch := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/5a35d699-5d11-4a5b-93ea-aa868cb1b75b/suspension", strings.NewReader(`{"suspended":false}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(patch, req)
	if patch.Code != http.StatusOK || uc.suspended == nil || *uc.suspended {
		t.Fatalf("patch status=%d suspended=%v", patch.Code, uc.suspended)
	}
}

func TestAdminUsersHandlerRejectsNonAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		auth.SetClaims(c, auth.Claims{UserID: "user", Role: rbac.RoleUser})
		c.Next()
	})
	router.GET("/api/v1/admin/users", middleware.RequireRole(rbac.RoleAdmin), NewAdminUsersHandler(&fakeAdminUsersUseCase{}).List)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

type fakeAdminUsersUseCase struct {
	listReq   dto.ListAdminUsersRequest
	userID    string
	suspended *bool
}

func (f *fakeAdminUsersUseCase) List(_ context.Context, _ auth.Claims, req dto.ListAdminUsersRequest) (dto.ListAdminUsersResponse, error) {
	f.listReq = req
	return dto.ListAdminUsersResponse{Items: []dto.AdminUserResponse{{ID: "id", Username: "ada", Email: "ada@example.test"}}}, nil
}
func (f *fakeAdminUsersUseCase) Get(_ context.Context, _ auth.Claims, params dto.UserIDRequest) (dto.AdminUserResponse, error) {
	f.userID = params.UserID
	return dto.AdminUserResponse{ID: params.UserID, Username: "ada", Email: "ada@example.test"}, nil
}
func (f *fakeAdminUsersUseCase) SetSuspension(_ context.Context, _ auth.Claims, _ dto.UserIDRequest, req dto.SetUserSuspensionRequest) (dto.AdminUserResponse, error) {
	f.suspended = req.Suspended
	return dto.AdminUserResponse{}, nil
}

func containsAny(value string, values ...string) bool {
	for _, candidate := range values {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}

var _ inbound.AdminUsersUseCase = (*fakeAdminUsersUseCase)(nil)
