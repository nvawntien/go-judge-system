package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/pkg/response"
	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/domain"

	"github.com/gin-gonic/gin"
)

type fakeGetAdminSubmissionDetailUseCase struct {
	response dto.GetAdminSubmissionDetailResponse
	err      error
	claims   auth.Claims
	req      dto.GetAdminSubmissionDetailRequest
	calls    int
}

func (f *fakeGetAdminSubmissionDetailUseCase) Execute(
	_ context.Context,
	claims auth.Claims,
	req dto.GetAdminSubmissionDetailRequest,
) (dto.GetAdminSubmissionDetailResponse, error) {
	f.calls++
	f.claims = claims
	f.req = req
	return f.response, f.err
}

func performGetAdminSubmissionDetailRequest(
	t *testing.T,
	target string,
	claims *auth.Claims,
	useCase *fakeGetAdminSubmissionDetailUseCase,
) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	handler := NewGetSubmissionDetailHandler(useCase)
	router := gin.New()
	router.GET("/api/v1/admin/submissions/:submission_id", func(c *gin.Context) {
		if claims != nil {
			auth.SetClaims(c, *claims)
		}
		handler.Handle(c)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	return recorder
}

func TestGetAdminSubmissionDetailHandlerBindsClaimsAndParams(t *testing.T) {
	claims := auth.Claims{UserID: "moderator", Username: "mod-name", Role: rbac.RoleModerator}
	useCase := &fakeGetAdminSubmissionDetailUseCase{
		response: dto.GetAdminSubmissionDetailResponse{
			ID:           77,
			ProblemID:    42,
			ProblemTitle: "Two Sum",
			UserID:       "user-123",
			Username:     "alice",
			Language:     "CPP",
			SourceCode:   "int main() { return 0; }\n",
			Status:       "ACCEPTED",
			CreatedAt:    time.Date(2026, 8, 2, 8, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Date(2026, 8, 2, 8, 1, 0, 0, time.UTC),
			TestResults:  []dto.AdminSubmissionTestResult{},
		},
	}

	recorder := performGetAdminSubmissionDetailRequest(
		t,
		"/api/v1/admin/submissions/77",
		&claims,
		useCase,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", recorder.Header().Get("Cache-Control"))
	}
	if useCase.calls != 1 {
		t.Fatalf("use case calls = %d, want 1", useCase.calls)
	}
	if useCase.claims != claims {
		t.Fatalf("claims = %+v, want %+v", useCase.claims, claims)
	}
	if useCase.req.SubmissionID != 77 {
		t.Fatalf("submission ID = %d, want 77", useCase.req.SubmissionID)
	}
}

func TestGetAdminSubmissionDetailHandlerRejectsMalformedID(t *testing.T) {
	claims := auth.Claims{UserID: "admin", Role: rbac.RoleAdmin}
	useCase := &fakeGetAdminSubmissionDetailUseCase{}
	recorder := performGetAdminSubmissionDetailRequest(
		t,
		"/api/v1/admin/submissions/not-a-number",
		&claims,
		useCase,
	)

	assertGetAdminSubmissionDetailError(t, recorder, http.StatusBadRequest, response.CodeParamInvalid)
	if useCase.calls != 0 {
		t.Fatalf("use case calls = %d, want 0", useCase.calls)
	}
}

func TestGetAdminSubmissionDetailHandlerUsesSharedErrorMapping(t *testing.T) {
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
			name: "not found", claims: &auth.Claims{UserID: "actor", Role: rbac.RoleModerator},
			err: domain.ErrSubmissionNotFound, wantStatus: http.StatusNotFound, wantCode: response.CodeNotFound, wantCalls: 1,
		},
		{
			name: "internal failure", claims: &auth.Claims{UserID: "actor", Role: rbac.RoleAdmin},
			err:        domain.ErrInternalServer.Wrap(errors.New("database unavailable")),
			wantStatus: http.StatusInternalServerError, wantCode: response.CodeInternalServer, wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase := &fakeGetAdminSubmissionDetailUseCase{err: tt.err}
			recorder := performGetAdminSubmissionDetailRequest(t, "/api/v1/admin/submissions/77", tt.claims, useCase)

			assertGetAdminSubmissionDetailError(t, recorder, tt.wantStatus, tt.wantCode)
			if useCase.calls != tt.wantCalls {
				t.Fatalf("use case calls = %d, want %d", useCase.calls, tt.wantCalls)
			}
		})
	}
}

func TestGetAdminSubmissionDetailHandlerSerializesSafeDetail(t *testing.T) {
	claims := auth.Claims{UserID: "admin", Role: rbac.RoleAdmin}
	runtime := 2
	memory := 128
	total := 1
	useCase := &fakeGetAdminSubmissionDetailUseCase{
		response: dto.GetAdminSubmissionDetailResponse{
			ID:                77,
			ProblemID:         42,
			ProblemTitle:      "Two Sum",
			UserID:            "user-123",
			Username:          "alice",
			Language:          "CPP",
			SourceCode:        "int main() { return 0; }\n",
			Status:            "ACCEPTED",
			CurrentAttemptID:  "attempt-current",
			PassedTestCount:   1,
			ExecutedTestCount: 1,
			TotalTestCount:    &total,
			RuntimeMS:         &runtime,
			MemoryKB:          &memory,
			TestResults: []dto.AdminSubmissionTestResult{
				{Index: 1, Status: "ACCEPTED", RuntimeMS: &runtime, MemoryKB: &memory},
			},
		},
	}
	recorder := performGetAdminSubmissionDetailRequest(t, "/api/v1/admin/submissions/77", &claims, useCase)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, required := range []string{
		`"source_code":"int main()`,
		`"current_attempt_id":"attempt-current"`,
		`"passed_test_count":1`,
		`"executed_test_count":1`,
		`"total_test_count":1`,
		`"test_results":[`,
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("body missing %s: %s", required, body)
		}
	}
	for _, forbidden := range []string{
		"actual_output",
		"expected_output",
		`"input"`,
		"object_key",
		"signed_url",
		"ticket",
		"container_id",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("body exposes %q: %s", forbidden, body)
		}
	}
}

func assertGetAdminSubmissionDetailError(
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
