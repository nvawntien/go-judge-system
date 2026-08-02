package user

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

type fakeIssueStreamTicketUseCase struct {
	response dto.IssueSubmissionStreamTicketResponse
	err      error
	claims   auth.Claims
	req      dto.IssueSubmissionStreamTicketRequest
	calls    int
}

func (f *fakeIssueStreamTicketUseCase) Execute(
	_ context.Context,
	claims auth.Claims,
	req dto.IssueSubmissionStreamTicketRequest,
) (dto.IssueSubmissionStreamTicketResponse, error) {
	f.calls++
	f.claims = claims
	f.req = req
	return f.response, f.err
}

func performIssueStreamTicketRequest(
	t *testing.T,
	path string,
	claims *auth.Claims,
	useCase *fakeIssueStreamTicketUseCase,
) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	handler := NewIssueSubmissionStreamTicketHandler(useCase)
	router := gin.New()
	router.POST("/api/v1/submissions/:submission_id/events/ticket", func(c *gin.Context) {
		if claims != nil {
			auth.SetClaims(c, *claims)
		}
		handler.Handle(c)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, nil)
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestIssueStreamTicketHandlerSuccess(t *testing.T) {
	expiresAt := time.Date(2026, 7, 30, 9, 2, 0, 0, time.UTC)
	claims := auth.Claims{UserID: "owner", Username: "owner-name", Role: rbac.RoleUser}
	useCase := &fakeIssueStreamTicketUseCase{
		response: dto.IssueSubmissionStreamTicketResponse{
			Ticket:    "opaque-ticket",
			ExpiresAt: expiresAt,
		},
	}

	recorder := performIssueStreamTicketRequest(
		t,
		"/api/v1/submissions/77/events/ticket",
		&claims,
		useCase,
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if useCase.calls != 1 || useCase.req.SubmissionID != 77 || useCase.claims != claims {
		t.Fatalf("use case calls/req/claims = %d/%+v/%+v", useCase.calls, useCase.req, useCase.claims)
	}

	var envelope struct {
		Status string                                  `json:"status"`
		Code   int                                     `json:"code"`
		Data   dto.IssueSubmissionStreamTicketResponse `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Status != "success" ||
		envelope.Code != response.CodeSuccess ||
		envelope.Data.Ticket != "opaque-ticket" ||
		!envelope.Data.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("envelope = %+v", envelope)
	}
}

func TestIssueStreamTicketHandlerRejectsInvalidIDAndMissingClaims(t *testing.T) {
	claims := auth.Claims{UserID: "owner", Role: rbac.RoleUser}
	tests := []struct {
		name       string
		path       string
		claims     *auth.Claims
		wantStatus int
		wantCode   int
	}{
		{name: "invalid id", path: "/api/v1/submissions/nope/events/ticket", claims: &claims, wantStatus: http.StatusBadRequest, wantCode: response.CodeParamInvalid},
		{name: "zero id", path: "/api/v1/submissions/0/events/ticket", claims: &claims, wantStatus: http.StatusBadRequest, wantCode: response.CodeParamInvalid},
		{name: "missing claims", path: "/api/v1/submissions/77/events/ticket", wantStatus: http.StatusUnauthorized, wantCode: response.CodeUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase := &fakeIssueStreamTicketUseCase{}
			recorder := performIssueStreamTicketRequest(t, tt.path, tt.claims, useCase)
			assertIssueStreamTicketError(t, recorder, tt.wantStatus, tt.wantCode)
			if useCase.calls != 0 {
				t.Fatalf("use case calls = %d, want 0", useCase.calls)
			}
		})
	}
}

func TestIssueStreamTicketHandlerMapsUseCaseErrors(t *testing.T) {
	claims := auth.Claims{UserID: "owner", Role: rbac.RoleUser}
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{name: "not found", err: domain.ErrSubmissionNotFound, wantStatus: http.StatusNotFound, wantCode: response.CodeNotFound},
		{name: "forbidden", err: domain.ErrSubmissionForbidden, wantStatus: http.StatusForbidden, wantCode: response.CodeForbidden},
		{name: "internal", err: domain.ErrInternalServer.Wrap(errors.New("database unavailable")), wantStatus: http.StatusInternalServerError, wantCode: response.CodeInternalServer},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase := &fakeIssueStreamTicketUseCase{err: tt.err}
			recorder := performIssueStreamTicketRequest(t, "/api/v1/submissions/77/events/ticket", &claims, useCase)
			assertIssueStreamTicketError(t, recorder, tt.wantStatus, tt.wantCode)
			if useCase.calls != 1 {
				t.Fatalf("use case calls = %d, want 1", useCase.calls)
			}
		})
	}
}

func assertIssueStreamTicketError(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	wantStatus int,
	wantCode int,
) {
	t.Helper()
	if recorder.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, wantStatus, recorder.Body.String())
	}
	var envelope response.APIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Status != "error" || envelope.Code != wantCode {
		t.Fatalf("envelope = %+v, want code %d", envelope, wantCode)
	}
}
