package user

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/pkg/response"
	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/domain"

	"github.com/gin-gonic/gin"
)

type fakeGetMyProfileStatsUseCase struct {
	response dto.GetMyProfileStatsResponse
	err      error
	claims   auth.Claims
	calls    int
}

func (f *fakeGetMyProfileStatsUseCase) Execute(_ context.Context, claims auth.Claims) (dto.GetMyProfileStatsResponse, error) {
	f.calls++
	f.claims = claims
	return f.response, f.err
}

func performGetMyProfileStatsRequest(t *testing.T, claims *auth.Claims, uc *fakeGetMyProfileStatsUseCase) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewGetMyProfileStatsHandler(uc)
	router.GET("/api/v1/me/profile-stats", func(c *gin.Context) {
		if claims != nil {
			auth.SetClaims(c, *claims)
		}
		handler.Handle(c)
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/me/profile-stats?user_id=other", nil))
	return recorder
}

func TestGetMyProfileStatsHandlerUsesAuthenticatedClaimsOnly(t *testing.T) {
	claims := auth.Claims{UserID: "actor", Role: rbac.RoleUser}
	uc := &fakeGetMyProfileStatsUseCase{response: dto.GetMyProfileStatsResponse{
		TotalSubmissions:     4,
		AttemptedProblems:    3,
		AcceptedSubmissions:  2,
		SolvedProblems:       2,
		VerdictDistribution:  []dto.ProfileStatsVerdictResponse{},
		LanguageDistribution: []dto.ProfileStatsLanguageResponse{},
		Activity:             []dto.ProfileStatsActivityResponse{},
	}}
	recorder := performGetMyProfileStatsRequest(t, &claims, uc)
	if recorder.Code != http.StatusOK || uc.calls != 1 || uc.claims != claims {
		t.Fatalf("status/calls/claims = %d/%d/%+v", recorder.Code, uc.calls, uc.claims)
	}
	var envelope struct {
		Status string                        `json:"status"`
		Code   int                           `json:"code"`
		Data   dto.GetMyProfileStatsResponse `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Status != "success" || envelope.Code != response.CodeSuccess || envelope.Data.AttemptedProblems != 3 || envelope.Data.Activity == nil {
		t.Fatalf("envelope = %+v", envelope)
	}
}

func TestGetMyProfileStatsHandlerMapsAuthAndRepositoryErrors(t *testing.T) {
	for _, tt := range []struct {
		name       string
		claims     *auth.Claims
		err        error
		wantStatus int
		wantCode   int
	}{
		{name: "missing claims", wantStatus: http.StatusUnauthorized, wantCode: response.CodeUnauthorized},
		{name: "repository error", claims: &auth.Claims{UserID: "actor", Role: rbac.RoleUser}, err: domain.ErrInternalServer.Wrap(errors.New("db")), wantStatus: http.StatusInternalServerError, wantCode: response.CodeInternalServer},
	} {
		t.Run(tt.name, func(t *testing.T) {
			recorder := performGetMyProfileStatsRequest(t, tt.claims, &fakeGetMyProfileStatsUseCase{err: tt.err})
			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
		})
	}
}
