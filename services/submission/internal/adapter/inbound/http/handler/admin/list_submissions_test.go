package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/pkg/response"
	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/domain"

	"github.com/gin-gonic/gin"
)

type fakeListAdminSubmissionsUseCase struct {
	response dto.ListAdminSubmissionsResponse
	err      error
	claims   auth.Claims
	req      dto.ListAdminSubmissionsRequest
	calls    int
}

func (f *fakeListAdminSubmissionsUseCase) Execute(
	_ context.Context,
	claims auth.Claims,
	req dto.ListAdminSubmissionsRequest,
) (dto.ListAdminSubmissionsResponse, error) {
	f.calls++
	f.claims = claims
	f.req = req
	return f.response, f.err
}

func performListAdminSubmissionsRequest(
	t *testing.T,
	target string,
	claims *auth.Claims,
	useCase *fakeListAdminSubmissionsUseCase,
) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	handler := NewListSubmissionsHandler(useCase)
	router := gin.New()
	router.GET("/api/v1/admin/submissions", func(c *gin.Context) {
		if claims != nil {
			auth.SetClaims(c, *claims)
		}
		handler.Handle(c)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	return recorder
}

func TestListAdminSubmissionsHandlerBindsClaimsAndQuery(t *testing.T) {
	page, limit, problemID := 2, 10, int64(42)
	claims := auth.Claims{UserID: "moderator", Username: "mod-name", Role: rbac.RoleModerator}
	useCase := &fakeListAdminSubmissionsUseCase{
		response: dto.ListAdminSubmissionsResponse{
			Items: []dto.AdminSubmissionListItem{},
			Pagination: dto.PaginationResponse{
				Page: 2, Limit: 10,
			},
		},
	}

	recorder := performListAdminSubmissionsRequest(
		t,
		"/api/v1/admin/submissions?page=2&limit=10&status=PENDING"+
			"&language=GO&problem_id=42&user_id=user-123&username=ignored",
		&claims,
		useCase,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if useCase.calls != 1 {
		t.Fatalf("use case calls = %d, want 1", useCase.calls)
	}
	if useCase.claims != claims {
		t.Fatalf("claims = %+v, want %+v", useCase.claims, claims)
	}
	req := useCase.req
	if req.Page == nil || *req.Page != page ||
		req.Limit == nil || *req.Limit != limit ||
		req.Status == nil || *req.Status != "PENDING" ||
		req.Language == nil || *req.Language != "GO" ||
		req.ProblemID == nil || *req.ProblemID != problemID ||
		req.UserID == nil || *req.UserID != "user-123" {
		t.Fatalf("bound request = %+v", req)
	}
}

func TestListAdminSubmissionsHandlerAllowsModeratorAndAdmin(t *testing.T) {
	for _, role := range []rbac.Role{rbac.RoleModerator, rbac.RoleAdmin} {
		t.Run(string(role), func(t *testing.T) {
			claims := auth.Claims{UserID: "actor", Role: role}
			useCase := &fakeListAdminSubmissionsUseCase{
				response: dto.ListAdminSubmissionsResponse{
					Items: []dto.AdminSubmissionListItem{},
					Pagination: dto.PaginationResponse{
						Page: 1, Limit: 20,
					},
				},
			}
			recorder := performListAdminSubmissionsRequest(t, "/api/v1/admin/submissions", &claims, useCase)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
			}
			if useCase.calls != 1 {
				t.Fatalf("use case calls = %d, want 1", useCase.calls)
			}
			if useCase.req.Page != nil || useCase.req.Limit != nil ||
				useCase.req.Status != nil || useCase.req.Language != nil ||
				useCase.req.ProblemID != nil || useCase.req.UserID != nil {
				t.Fatalf("default request should contain nil optional fields: %+v", useCase.req)
			}
		})
	}
}

func TestListAdminSubmissionsHandlerRejectsMalformedQuery(t *testing.T) {
	claims := auth.Claims{UserID: "admin", Role: rbac.RoleAdmin}
	for _, target := range []string{
		"/api/v1/admin/submissions?page=invalid",
		"/api/v1/admin/submissions?limit=invalid",
		"/api/v1/admin/submissions?problem_id=invalid",
	} {
		t.Run(target, func(t *testing.T) {
			useCase := &fakeListAdminSubmissionsUseCase{}
			recorder := performListAdminSubmissionsRequest(t, target, &claims, useCase)

			assertListAdminSubmissionsError(t, recorder, http.StatusBadRequest, response.CodeBadRequest)
			if useCase.calls != 0 {
				t.Fatalf("use case calls = %d, want 0", useCase.calls)
			}
		})
	}
}

func TestListAdminSubmissionsHandlerUsesSharedErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		claims     *auth.Claims
		err        error
		wantStatus int
		wantCode   int
		wantCalls  int
	}{
		{name: "missing claims", wantStatus: http.StatusUnauthorized, wantCode: response.CodeUnauthorized},
		{
			name: "user forbidden", claims: &auth.Claims{UserID: "actor", Role: rbac.RoleUser},
			err: domain.ErrSubmissionForbidden, wantStatus: http.StatusForbidden, wantCode: response.CodeForbidden, wantCalls: 1,
		},
		{
			name: "contributor forbidden", claims: &auth.Claims{UserID: "actor", Role: rbac.RoleContributor},
			err: domain.ErrSubmissionForbidden, wantStatus: http.StatusForbidden, wantCode: response.CodeForbidden, wantCalls: 1,
		},
		{
			name: "invalid filter", claims: &auth.Claims{UserID: "actor", Role: rbac.RoleAdmin},
			err: domain.ErrInvalidUserID, wantStatus: http.StatusBadRequest, wantCode: response.CodeBadRequest, wantCalls: 1,
		},
		{
			name: "internal failure", claims: &auth.Claims{UserID: "actor", Role: rbac.RoleAdmin},
			err:        domain.ErrInternalServer.Wrap(errors.New("database unavailable")),
			wantStatus: http.StatusInternalServerError, wantCode: response.CodeInternalServer, wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase := &fakeListAdminSubmissionsUseCase{err: tt.err}
			recorder := performListAdminSubmissionsRequest(t, "/api/v1/admin/submissions", tt.claims, useCase)

			assertListAdminSubmissionsError(t, recorder, tt.wantStatus, tt.wantCode)
			if useCase.calls != tt.wantCalls {
				t.Fatalf("use case calls = %d, want %d", useCase.calls, tt.wantCalls)
			}
		})
	}
}

func TestListAdminSubmissionsHandlerSerializesCompactList(t *testing.T) {
	claims := auth.Claims{UserID: "admin", Role: rbac.RoleAdmin}
	useCase := &fakeListAdminSubmissionsUseCase{
		response: dto.ListAdminSubmissionsResponse{
			Items: []dto.AdminSubmissionListItem{
				{
					ID: 123, ProblemID: 42, ProblemTitle: "Two Sum",
					UserID: "user-123", Username: "vantien",
					Language: "GO", Status: "PENDING",
				},
			},
			Pagination: dto.PaginationResponse{Page: 1, Limit: 20, Total: 1, TotalPages: 1},
		},
	}
	recorder := performListAdminSubmissionsRequest(t, "/api/v1/admin/submissions", &claims, useCase)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, required := range []string{
		`"user_id":"user-123"`,
		`"username":"vantien"`,
		`"problem_title":"Two Sum"`,
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("body missing %s: %s", required, body)
		}
	}
	for _, forbidden := range []string{"source_code", "problem_name", "role"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("body exposes %q: %s", forbidden, body)
		}
	}
}

func TestListAdminSubmissionsHandlerSerializesNonNilEmptyList(t *testing.T) {
	claims := auth.Claims{UserID: "admin", Role: rbac.RoleAdmin}
	useCase := &fakeListAdminSubmissionsUseCase{
		response: dto.ListAdminSubmissionsResponse{
			Items:      []dto.AdminSubmissionListItem{},
			Pagination: dto.PaginationResponse{Page: 1, Limit: 20},
		},
	}
	recorder := performListAdminSubmissionsRequest(t, "/api/v1/admin/submissions", &claims, useCase)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"items":[]`) {
		t.Fatalf("body must contain non-null empty items array: %s", recorder.Body.String())
	}

	var envelope struct {
		Status string                           `json:"status"`
		Code   int                              `json:"code"`
		Data   dto.ListAdminSubmissionsResponse `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Status != "success" || envelope.Code != response.CodeSuccess || envelope.Data.Items == nil {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
}

func assertListAdminSubmissionsError(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	wantStatus int,
	wantCode int,
) {
	t.Helper()
	if recorder.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, wantStatus, recorder.Body.String())
	}

	var envelope struct {
		Status string `json:"status"`
		Code   int    `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Status != "error" || envelope.Code != wantCode {
		t.Fatalf("unexpected error envelope: %+v", envelope)
	}
}
