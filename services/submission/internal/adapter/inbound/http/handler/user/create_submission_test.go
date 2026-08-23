package user

import (
	"bytes"
	"context"
	"encoding/json"
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

type fakeCreateSubmissionUseCase struct {
	claims auth.Claims
	req    dto.CreateSubmissionRequest
	called bool
	err    error
}

func (f *fakeCreateSubmissionUseCase) Execute(
	_ context.Context,
	claims auth.Claims,
	req dto.CreateSubmissionRequest,
) (dto.CreateSubmissionResponse, error) {
	f.called = true
	f.claims = claims
	f.req = req
	if f.err != nil {
		return dto.CreateSubmissionResponse{}, f.err
	}
	return dto.CreateSubmissionResponse{
		ID:           77,
		ProblemID:    req.ProblemID,
		ProblemTitle: "Two Sum",
		Language:     req.Language,
		Status:       "PENDING",
		CreatedAt:    time.Date(2026, 7, 22, 1, 2, 3, 0, time.UTC),
	}, nil
}

func performRequest(t *testing.T, body string, claims *auth.Claims) (*httptest.ResponseRecorder, *fakeCreateSubmissionUseCase) {
	t.Helper()
	return performRequestWithUseCase(t, body, claims, &fakeCreateSubmissionUseCase{})
}

func performRequestWithUseCase(
	t *testing.T,
	body string,
	claims *auth.Claims,
	useCase *fakeCreateSubmissionUseCase,
) (*httptest.ResponseRecorder, *fakeCreateSubmissionUseCase) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	handler := NewCreateSubmissionHandler(useCase)
	router := gin.New()
	router.POST("/api/v1/submissions", func(c *gin.Context) {
		if claims != nil {
			auth.SetClaims(c, *claims)
		}
		handler.Handle(c)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)
	return recorder, useCase
}

func TestCreateSubmissionHandler_Success(t *testing.T) {
	claims := auth.Claims{UserID: "trusted-user", Username: "trusted-name", Role: rbac.RoleContributor}
	recorder, useCase := performRequest(t, `{
		"problem_id": 42,
		"problem_title": "Attacker Title",
		"language": "GO",
		"source_code": "package main",
		"user_id": "attacker",
		"username": "attacker",
		"status": "ACCEPTED",
		"actor_user_id": "attacker",
		"actor_role": "admin"
	}`, &claims)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if !useCase.called {
		t.Fatal("use case was not called")
	}
	if useCase.claims.UserID != claims.UserID || useCase.claims.Username != claims.Username || useCase.claims.Role != claims.Role {
		t.Fatalf("claims = %+v, want trusted claims %+v", useCase.claims, claims)
	}
	if useCase.req.ProblemID != 42 || useCase.req.Language != "GO" || useCase.req.SourceCode != "package main" {
		t.Fatalf("request = %+v", useCase.req)
	}

	var responseBody struct {
		Status string                       `json:"status"`
		Code   int                          `json:"code"`
		Msg    string                       `json:"msg"`
		Data   dto.CreateSubmissionResponse `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if responseBody.Status != "success" || responseBody.Code != 20100 || responseBody.Msg != "" {
		t.Fatalf("unexpected envelope: %+v", responseBody)
	}
	if responseBody.Data.ID != 77 ||
		responseBody.Data.ProblemID != 42 ||
		responseBody.Data.ProblemTitle != "Two Sum" ||
		responseBody.Data.Language != "GO" ||
		responseBody.Data.Status != "PENDING" ||
		!responseBody.Data.CreatedAt.Equal(time.Date(2026, 7, 22, 1, 2, 3, 0, time.UTC)) {
		t.Fatalf("unexpected response data: %+v", responseBody.Data)
	}
}

func TestCreateSubmissionHandlerProblemErrors(t *testing.T) {
	claims := auth.Claims{UserID: "user-1"}
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{name: "inaccessible hidden problem", err: domain.ErrProblemNotFound, wantStatus: http.StatusNotFound, wantCode: 40400},
		{name: "missing actor authority", err: domain.ErrProblemActorUnauthenticated, wantStatus: http.StatusUnauthorized, wantCode: 40100},
		{name: "unsupported actor authority", err: domain.ErrProblemActorForbidden, wantStatus: http.StatusForbidden, wantCode: 40300},
		{name: "problem service unavailable", err: domain.ErrProblemServiceUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: 50300},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder, useCase := performRequestWithUseCase(
				t,
				`{"problem_id":42,"language":"GO","source_code":"x"}`,
				&claims,
				&fakeCreateSubmissionUseCase{err: tt.err},
			)
			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			if !useCase.called {
				t.Fatal("use case was not called")
			}

			var responseBody struct {
				Status string `json:"status"`
				Code   int    `json:"code"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &responseBody); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if responseBody.Status != "error" || responseBody.Code != tt.wantCode {
				t.Fatalf("unexpected envelope: %+v", responseBody)
			}
		})
	}
}

func TestCreateSubmissionHandler_MalformedJSON(t *testing.T) {
	claims := auth.Claims{UserID: "user-1"}
	recorder, useCase := performRequest(t, `{"problem_id":`, &claims)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if useCase.called {
		t.Fatal("use case must not be called for malformed JSON")
	}
}

func TestCreateSubmissionHandler_MissingClaims(t *testing.T) {
	recorder, useCase := performRequest(t, `{"problem_id":1,"language":"GO","source_code":"x"}`, nil)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if useCase.called {
		t.Fatal("use case must not be called without claims")
	}
}

func TestCreateSubmissionHandler_InvalidPayload(t *testing.T) {
	claims := auth.Claims{UserID: "user-1"}
	recorder, useCase := performRequest(t, `{"problem_id":0,"language":"GO","source_code":"x"}`, &claims)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if useCase.called {
		t.Fatal("use case must not be called for invalid payload")
	}
}

func TestCreateSubmissionHandler_CooldownResponse(t *testing.T) {
	claims := auth.Claims{UserID: "user-1"}
	recorder, useCase := performRequestWithUseCase(
		t,
		`{"problem_id":42,"language":"GO","source_code":"x"}`,
		&claims,
		&fakeCreateSubmissionUseCase{err: response.NewRateLimitError("You're submitting too quickly. Please try again shortly.", 2*time.Second)},
	)

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusTooManyRequests, recorder.Body.String())
	}
	if got := recorder.Header().Get("Retry-After"); got != "2" {
		t.Fatalf("Retry-After = %q, want 2", got)
	}
	if !useCase.called {
		t.Fatal("use case was not called")
	}

	var responseBody struct {
		Status string `json:"status"`
		Code   int    `json:"code"`
		Msg    string `json:"msg"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if responseBody.Status != "error" || responseBody.Code != 42900 || responseBody.Msg != "You're submitting too quickly. Please try again shortly." {
		t.Fatalf("unexpected rate-limit envelope: %+v", responseBody)
	}
}
