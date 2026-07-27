package user

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

type fakeListMySubmissionsUseCase struct {
	response dto.ListMySubmissionsResponse
	err      error
	claims   auth.Claims
	req      dto.ListMySubmissionsRequest
	calls    int
}

func (f *fakeListMySubmissionsUseCase) Execute(
	_ context.Context,
	claims auth.Claims,
	req dto.ListMySubmissionsRequest,
) (dto.ListMySubmissionsResponse, error) {
	f.calls++
	f.claims = claims
	f.req = req
	return f.response, f.err
}

func performListMySubmissionsRequest(
	t *testing.T,
	target string,
	claims *auth.Claims,
	useCase *fakeListMySubmissionsUseCase,
) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	handler := NewListMySubmissionsHandler(useCase)
	router := gin.New()
	router.GET("/api/v1/me/submissions", func(c *gin.Context) {
		if claims != nil {
			auth.SetClaims(c, *claims)
		}
		handler.Handle(c)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, target, nil),
	)
	return recorder
}

func TestListMySubmissionsHandlerBindsClaimsAndQuery(t *testing.T) {
	page, limit, problemID := 2, 10, int64(42)
	tests := []struct {
		name   string
		target string
		assert func(*testing.T, dto.ListMySubmissionsRequest)
	}{
		{
			name:   "default query",
			target: "/api/v1/me/submissions",
			assert: func(t *testing.T, req dto.ListMySubmissionsRequest) {
				if req.Page != nil || req.Limit != nil {
					t.Fatalf("absent pagination must remain nil: %+v", req)
				}
			},
		},
		{
			name:   "custom pagination",
			target: "/api/v1/me/submissions?page=2&limit=10",
			assert: func(t *testing.T, req dto.ListMySubmissionsRequest) {
				if req.Page == nil || *req.Page != page || req.Limit == nil || *req.Limit != limit {
					t.Fatalf("pagination = %+v", req)
				}
			},
		},
		{
			name:   "status filter",
			target: "/api/v1/me/submissions?status=ACCEPTED",
			assert: func(t *testing.T, req dto.ListMySubmissionsRequest) {
				if req.Status != "ACCEPTED" {
					t.Fatalf("status = %q", req.Status)
				}
			},
		},
		{
			name:   "language filter",
			target: "/api/v1/me/submissions?language=GO",
			assert: func(t *testing.T, req dto.ListMySubmissionsRequest) {
				if req.Language != "GO" {
					t.Fatalf("language = %q", req.Language)
				}
			},
		},
		{
			name:   "problem filter",
			target: "/api/v1/me/submissions?problem_id=42",
			assert: func(t *testing.T, req dto.ListMySubmissionsRequest) {
				if req.ProblemID == nil || *req.ProblemID != problemID {
					t.Fatalf("problem ID = %v", req.ProblemID)
				}
			},
		},
		{
			name: "combined filters cannot override actor",
			target: "/api/v1/me/submissions?page=2&limit=10&status=PENDING" +
				"&language=CPP&problem_id=42&user_id=attacker&actor_role=admin",
			assert: func(t *testing.T, req dto.ListMySubmissionsRequest) {
				if req.Page == nil || *req.Page != page ||
					req.Limit == nil || *req.Limit != limit ||
					req.Status != "PENDING" ||
					req.Language != "CPP" ||
					req.ProblemID == nil || *req.ProblemID != problemID {
					t.Fatalf("combined request = %+v", req)
				}
			},
		},
	}

	claims := auth.Claims{
		UserID:   "verified-actor",
		Username: "verified-name",
		Role:     rbac.RoleModerator,
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase := &fakeListMySubmissionsUseCase{
				response: dto.ListMySubmissionsResponse{
					Items: []dto.SubmissionListItem{},
					Pagination: dto.PaginationResponse{
						Page: 1, Limit: 20,
					},
				},
			}
			recorder := performListMySubmissionsRequest(t, tt.target, &claims, useCase)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
			}
			if useCase.calls != 1 {
				t.Fatalf("use case calls = %d, want 1", useCase.calls)
			}
			if useCase.claims.UserID != claims.UserID || useCase.claims.Role != claims.Role {
				t.Fatalf("claims = %+v, want verified claims %+v", useCase.claims, claims)
			}
			tt.assert(t, useCase.req)
		})
	}
}

func TestListMySubmissionsHandlerRejectsMalformedQuery(t *testing.T) {
	claims := auth.Claims{UserID: "actor", Role: rbac.RoleUser}
	for _, target := range []string{
		"/api/v1/me/submissions?page=invalid",
		"/api/v1/me/submissions?limit=invalid",
		"/api/v1/me/submissions?problem_id=invalid",
	} {
		t.Run(target, func(t *testing.T) {
			useCase := &fakeListMySubmissionsUseCase{}
			recorder := performListMySubmissionsRequest(t, target, &claims, useCase)

			assertListMySubmissionsError(t, recorder, http.StatusBadRequest, response.CodeBadRequest)
			if useCase.calls != 0 {
				t.Fatalf("use case calls = %d, want 0", useCase.calls)
			}
		})
	}
}

func TestListMySubmissionsHandlerUsesSharedErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		claims     *auth.Claims
		err        error
		wantStatus int
		wantCode   int
		wantCalls  int
	}{
		{
			name:       "missing claims",
			wantStatus: http.StatusUnauthorized,
			wantCode:   response.CodeUnauthorized,
		},
		{
			name:       "invalid status",
			claims:     &auth.Claims{UserID: "actor", Role: rbac.RoleUser},
			err:        domain.ErrInvalidSubmissionStatus,
			wantStatus: http.StatusBadRequest,
			wantCode:   response.CodeBadRequest,
			wantCalls:  1,
		},
		{
			name:       "invalid language",
			claims:     &auth.Claims{UserID: "actor", Role: rbac.RoleUser},
			err:        domain.ErrInvalidLanguage,
			wantStatus: http.StatusBadRequest,
			wantCode:   response.CodeBadRequest,
			wantCalls:  1,
		},
		{
			name:       "unsupported role",
			claims:     &auth.Claims{UserID: "actor", Role: rbac.Role("auditor")},
			err:        domain.ErrSubmissionForbidden,
			wantStatus: http.StatusForbidden,
			wantCode:   response.CodeForbidden,
			wantCalls:  1,
		},
		{
			name:       "internal failure",
			claims:     &auth.Claims{UserID: "actor", Role: rbac.RoleUser},
			err:        domain.ErrInternalServer.Wrap(errors.New("database unavailable")),
			wantStatus: http.StatusInternalServerError,
			wantCode:   response.CodeInternalServer,
			wantCalls:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase := &fakeListMySubmissionsUseCase{err: tt.err}
			recorder := performListMySubmissionsRequest(
				t,
				"/api/v1/me/submissions",
				tt.claims,
				useCase,
			)

			assertListMySubmissionsError(t, recorder, tt.wantStatus, tt.wantCode)
			if useCase.calls != tt.wantCalls {
				t.Fatalf("use case calls = %d, want %d", useCase.calls, tt.wantCalls)
			}
		})
	}
}

func TestListMySubmissionsHandlerSerializesCompactEmptyList(t *testing.T) {
	claims := auth.Claims{UserID: "actor", Role: rbac.RoleAdmin}
	useCase := &fakeListMySubmissionsUseCase{
		response: dto.ListMySubmissionsResponse{
			Items: []dto.SubmissionListItem{},
			Pagination: dto.PaginationResponse{
				Page: 1, Limit: 20,
			},
		},
	}
	recorder := performListMySubmissionsRequest(
		t,
		"/api/v1/me/submissions",
		&claims,
		useCase,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"items":[]`) {
		t.Fatalf("body must contain a non-null empty items array: %s", body)
	}
	if strings.Contains(body, "source_code") || strings.Contains(body, "problem_name") {
		t.Fatalf("body exposes forbidden list fields: %s", body)
	}

	var envelope struct {
		Status string                        `json:"status"`
		Code   int                           `json:"code"`
		Data   dto.ListMySubmissionsResponse `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Status != "success" || envelope.Code != response.CodeSuccess {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
}

func TestListMySubmissionsHandlerUsesPublicListFieldNames(t *testing.T) {
	claims := auth.Claims{UserID: "actor", Role: rbac.RoleUser}
	useCase := &fakeListMySubmissionsUseCase{
		response: dto.ListMySubmissionsResponse{
			Items: []dto.SubmissionListItem{
				{
					ID:           123,
					ProblemID:    42,
					ProblemTitle: "Two Sum",
					Language:     "GO",
					Status:       "PENDING",
				},
			},
			Pagination: dto.PaginationResponse{
				Page: 1, Limit: 20, Total: 1, TotalPages: 1,
			},
		},
	}
	recorder := performListMySubmissionsRequest(
		t,
		"/api/v1/me/submissions",
		&claims,
		useCase,
	)

	body := recorder.Body.String()
	if !strings.Contains(body, `"problem_title":"Two Sum"`) {
		t.Fatalf("body does not use problem_title: %s", body)
	}
	for _, forbidden := range []string{
		"problem_name",
		"source_code",
		"user_id",
		"username",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("body exposes %q: %s", forbidden, body)
		}
	}
}

func assertListMySubmissionsError(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	wantStatus int,
	wantCode int,
) {
	t.Helper()
	if recorder.Code != wantStatus {
		t.Fatalf(
			"status = %d, want %d; body=%s",
			recorder.Code,
			wantStatus,
			recorder.Body.String(),
		)
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
