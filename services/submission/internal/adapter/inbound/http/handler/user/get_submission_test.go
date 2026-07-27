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

type fakeGetSubmissionUseCase struct {
	response dto.GetSubmissionResponse
	err      error
	claims   auth.Claims
	req      dto.GetSubmissionRequest
	calls    int
}

func (f *fakeGetSubmissionUseCase) Execute(
	_ context.Context,
	claims auth.Claims,
	req dto.GetSubmissionRequest,
) (dto.GetSubmissionResponse, error) {
	f.calls++
	f.claims = claims
	f.req = req
	return f.response, f.err
}

func getSubmissionResponseFixture() dto.GetSubmissionResponse {
	return dto.GetSubmissionResponse{
		ID:           123,
		ProblemID:    42,
		ProblemTitle: "Two Sum",
		UserID:       "owner",
		Username:     "owner-name",
		Language:     "GO",
		SourceCode:   "package main\n",
		Status:       "PENDING",
		CreatedAt:    time.Date(2026, 7, 23, 14, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 7, 23, 14, 1, 0, 0, time.UTC),
	}
}

func performGetSubmissionRequest(
	t *testing.T,
	path string,
	claims *auth.Claims,
	useCase *fakeGetSubmissionUseCase,
) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	handler := NewGetSubmissionHandler(useCase)
	router := gin.New()
	router.GET("/api/v1/submissions/:submission_id", func(c *gin.Context) {
		if claims != nil {
			auth.SetClaims(c, *claims)
		}
		handler.Handle(c)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestGetSubmissionHandlerSuccess(t *testing.T) {
	tests := []struct {
		name string
		role rbac.Role
	}{
		{name: "owner", role: rbac.RoleUser},
		{name: "moderator", role: rbac.RoleModerator},
		{name: "admin", role: rbac.RoleAdmin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := auth.Claims{UserID: "actor", Username: "actor-name", Role: tt.role}
			useCase := &fakeGetSubmissionUseCase{response: getSubmissionResponseFixture()}
			recorder := performGetSubmissionRequest(t, "/api/v1/submissions/123", &claims, useCase)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
			}
			if useCase.calls != 1 {
				t.Fatalf("use case calls = %d, want 1", useCase.calls)
			}
			wantRequest := dto.GetSubmissionRequest{SubmissionID: 123}
			if useCase.req != wantRequest {
				t.Fatalf("request = %+v, want %+v", useCase.req, wantRequest)
			}
			if useCase.claims != claims {
				t.Fatalf("claims = %+v, want %+v", useCase.claims, claims)
			}

			var responseBody struct {
				Status string                    `json:"status"`
				Code   int                       `json:"code"`
				Msg    string                    `json:"msg"`
				Data   dto.GetSubmissionResponse `json:"data"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &responseBody); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if responseBody.Status != "success" || responseBody.Code != response.CodeSuccess || responseBody.Msg != "" {
				t.Fatalf("unexpected envelope: %+v", responseBody)
			}
			if responseBody.Data != getSubmissionResponseFixture() {
				t.Fatalf("response data = %+v, want %+v", responseBody.Data, getSubmissionResponseFixture())
			}
		})
	}
}

func TestGetSubmissionHandlerRejectsInvalidSubmissionID(t *testing.T) {
	claims := auth.Claims{UserID: "actor", Role: rbac.RoleUser}
	for _, path := range []string{
		"/api/v1/submissions/not-a-number",
		"/api/v1/submissions/0",
		"/api/v1/submissions/-1",
	} {
		t.Run(path, func(t *testing.T) {
			useCase := &fakeGetSubmissionUseCase{}
			recorder := performGetSubmissionRequest(t, path, &claims, useCase)

			assertGetSubmissionError(t, recorder, http.StatusBadRequest, response.CodeParamInvalid)
			if useCase.calls != 0 {
				t.Fatalf("use case calls = %d, want 0", useCase.calls)
			}
		})
	}
}

func TestGetSubmissionHandlerRejectsMissingSubmissionID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	useCase := &fakeGetSubmissionUseCase{}
	handler := NewGetSubmissionHandler(useCase)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/submissions", nil)
	auth.SetClaims(c, auth.Claims{UserID: "actor", Role: rbac.RoleUser})

	handler.Handle(c)

	assertGetSubmissionError(t, recorder, http.StatusBadRequest, response.CodeParamInvalid)
	if useCase.calls != 0 {
		t.Fatalf("use case calls = %d, want 0", useCase.calls)
	}
}

func TestGetSubmissionHandlerMissingClaims(t *testing.T) {
	useCase := &fakeGetSubmissionUseCase{}
	recorder := performGetSubmissionRequest(t, "/api/v1/submissions/123", nil, useCase)

	assertGetSubmissionError(t, recorder, http.StatusUnauthorized, response.CodeUnauthorized)
	if useCase.calls != 0 {
		t.Fatalf("use case calls = %d, want 0", useCase.calls)
	}
}

func TestGetSubmissionHandlerMapsUseCaseErrors(t *testing.T) {
	claims := auth.Claims{UserID: "actor", Role: rbac.RoleUser}
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{name: "inaccessible submission", err: domain.ErrSubmissionNotFound, wantStatus: http.StatusNotFound, wantCode: response.CodeNotFound},
		{name: "missing submission", err: domain.ErrSubmissionNotFound, wantStatus: http.StatusNotFound, wantCode: response.CodeNotFound},
		{name: "unsupported role", err: domain.ErrSubmissionForbidden, wantStatus: http.StatusForbidden, wantCode: response.CodeForbidden},
		{
			name:       "repository failure",
			err:        domain.ErrInternalServer.Wrap(errors.New("database unavailable")),
			wantStatus: http.StatusInternalServerError,
			wantCode:   response.CodeInternalServer,
		},
	}

	var notFoundBodies []string
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase := &fakeGetSubmissionUseCase{err: tt.err}
			recorder := performGetSubmissionRequest(t, "/api/v1/submissions/123", &claims, useCase)

			assertGetSubmissionError(t, recorder, tt.wantStatus, tt.wantCode)
			if useCase.calls != 1 {
				t.Fatalf("use case calls = %d, want 1", useCase.calls)
			}
			if tt.wantStatus == http.StatusNotFound {
				notFoundBodies = append(notFoundBodies, recorder.Body.String())
			}
		})
	}

	if len(notFoundBodies) != 2 || notFoundBodies[0] != notFoundBodies[1] {
		t.Fatalf("inaccessible and missing responses differ: %q / %q", notFoundBodies[0], notFoundBodies[1])
	}
}

func assertGetSubmissionError(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	wantStatus int,
	wantCode int,
) {
	t.Helper()
	if recorder.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, wantStatus, recorder.Body.String())
	}

	var responseBody struct {
		Status string `json:"status"`
		Code   int    `json:"code"`
		Msg    string `json:"msg"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if responseBody.Status != "error" || responseBody.Code != wantCode {
		t.Fatalf("unexpected error envelope: %+v", responseBody)
	}
	if wantStatus == http.StatusInternalServerError && responseBody.Msg != "internal server error" {
		t.Fatalf("internal response leaked details: %+v", responseBody)
	}
}
