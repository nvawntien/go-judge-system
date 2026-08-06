package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/pkg/response"
	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/domain"

	"github.com/gin-gonic/gin"
)

type fakeRejudgeSubmissionUseCase struct {
	response dto.RejudgeAdminSubmissionResponse
	err      error
	claims   auth.Claims
	req      dto.RejudgeAdminSubmissionRequest
	calls    int
}

func (f *fakeRejudgeSubmissionUseCase) Execute(
	_ context.Context,
	claims auth.Claims,
	req dto.RejudgeAdminSubmissionRequest,
) (dto.RejudgeAdminSubmissionResponse, error) {
	f.calls++
	f.claims = claims
	f.req = req
	return f.response, f.err
}

func performRejudgeSubmissionRequest(
	t *testing.T,
	target string,
	claims *auth.Claims,
	useCase *fakeRejudgeSubmissionUseCase,
) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	handler := NewRejudgeSubmissionHandler(useCase)
	router := gin.New()
	router.POST("/api/v1/admin/submissions/:submission_id/rejudge", func(c *gin.Context) {
		if claims != nil {
			auth.SetClaims(c, *claims)
		}
		handler.Handle(c)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, target, nil))
	return recorder
}

func TestRejudgeSubmissionHandlerBindsClaimsAndParams(t *testing.T) {
	claims := auth.Claims{UserID: "moderator", Username: "mod-name", Role: rbac.RoleModerator}
	useCase := &fakeRejudgeSubmissionUseCase{
		response: dto.RejudgeAdminSubmissionResponse{
			SubmissionID:             77,
			AttemptID:                "attempt-new",
			Status:                   "PENDING",
			AttemptTrigger:           "ADMIN_REJUDGE",
			AttemptTriggeredByUserID: "moderator",
			EnqueuedAt:               time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC),
		},
	}

	recorder := performRejudgeSubmissionRequest(t, "/api/v1/admin/submissions/77/rejudge", &claims, useCase)

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

func TestRejudgeSubmissionHandlerRejectsMalformedID(t *testing.T) {
	claims := auth.Claims{UserID: "admin", Role: rbac.RoleAdmin}
	useCase := &fakeRejudgeSubmissionUseCase{}
	recorder := performRejudgeSubmissionRequest(
		t,
		"/api/v1/admin/submissions/not-a-number/rejudge",
		&claims,
		useCase,
	)

	assertRejudgeSubmissionError(t, recorder, http.StatusBadRequest, response.CodeParamInvalid)
	if useCase.calls != 0 {
		t.Fatalf("use case calls = %d, want 0", useCase.calls)
	}
}

func TestRejudgeSubmissionHandlerUsesSharedErrorMapping(t *testing.T) {
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
			name: "forbidden", claims: &auth.Claims{UserID: "actor", Role: rbac.RoleUser},
			err: domain.ErrSubmissionForbidden, wantStatus: http.StatusForbidden, wantCode: response.CodeForbidden, wantCalls: 1,
		},
		{
			name: "not found", claims: &auth.Claims{UserID: "actor", Role: rbac.RoleModerator},
			err: domain.ErrSubmissionNotFound, wantStatus: http.StatusNotFound, wantCode: response.CodeNotFound, wantCalls: 1,
		},
		{
			name: "conflict active submission", claims: &auth.Claims{UserID: "actor", Role: rbac.RoleModerator},
			err: domain.ErrSubmissionRejudgeConflict, wantStatus: http.StatusConflict, wantCode: response.CodeConflict, wantCalls: 1,
		},
		{
			name: "conflict missing testcase", claims: &auth.Claims{UserID: "actor", Role: rbac.RoleAdmin},
			err: domain.ErrSubmissionTestCaseRequired, wantStatus: http.StatusConflict, wantCode: response.CodeConflict, wantCalls: 1,
		},
		{
			name: "internal failure", claims: &auth.Claims{UserID: "actor", Role: rbac.RoleAdmin},
			err:        domain.ErrInternalServer.Wrap(errors.New("database unavailable")),
			wantStatus: http.StatusInternalServerError, wantCode: response.CodeInternalServer, wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase := &fakeRejudgeSubmissionUseCase{err: tt.err}
			recorder := performRejudgeSubmissionRequest(t, "/api/v1/admin/submissions/77/rejudge", tt.claims, useCase)

			assertRejudgeSubmissionError(t, recorder, tt.wantStatus, tt.wantCode)
			if useCase.calls != tt.wantCalls {
				t.Fatalf("use case calls = %d, want %d", useCase.calls, tt.wantCalls)
			}
		})
	}
}

func assertRejudgeSubmissionError(
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
