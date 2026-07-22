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
		ID:        77,
		ProblemID: req.ProblemID,
		Language:  req.Language,
		Status:    "PENDING",
		CreatedAt: time.Date(2026, 7, 22, 1, 2, 3, 0, time.UTC),
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
	claims := auth.Claims{UserID: "trusted-user", Username: "trusted-name"}
	recorder, useCase := performRequest(t, `{
		"problem_id": 42,
		"language": "GO",
		"source_code": "package main",
		"user_id": "attacker",
		"username": "attacker",
		"status": "ACCEPTED"
	}`, &claims)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if !useCase.called {
		t.Fatal("use case was not called")
	}
	if useCase.claims.UserID != claims.UserID || useCase.claims.Username != claims.Username {
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
	if responseBody.Data.ID != 77 || responseBody.Data.Status != "PENDING" {
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
		{name: "problem not found", err: domain.ErrProblemNotFound, wantStatus: http.StatusNotFound, wantCode: 40400},
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
