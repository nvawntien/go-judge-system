package user

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/domain"

	"github.com/gin-gonic/gin"
)

type fakeGetPublicProfileStatsUseCase struct {
	response dto.GetMyProfileStatsResponse
	err      error
	req      dto.GetPublicProfileStatsRequest
}

func (f *fakeGetPublicProfileStatsUseCase) Execute(_ context.Context, req dto.GetPublicProfileStatsRequest) (dto.GetMyProfileStatsResponse, error) {
	f.req = req
	return f.response, f.err
}

func TestGetPublicProfileStatsHandlerIsPublicAndBindsOnlyUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := &fakeGetPublicProfileStatsUseCase{response: dto.GetMyProfileStatsResponse{SolvedProblems: 2}}
	router := gin.New()
	router.GET("/api/v1/users/:username/profile-stats", NewGetPublicProfileStatsHandler(uc).Handle)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/users/ada/profile-stats?user_id=attacker", nil))
	if recorder.Code != http.StatusOK || uc.req.Username != "ada" {
		t.Fatalf("status/request = %d/%+v; body=%s", recorder.Code, uc.req, recorder.Body.String())
	}
}

func TestGetPublicProfileStatsHandlerMapsPublicErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		err    error
		status int
	}{
		{domain.ErrPublicProfileNotFound, http.StatusNotFound},
		{domain.ErrAuthServiceUnavailable, http.StatusServiceUnavailable},
	} {
		router := gin.New()
		router.GET("/api/v1/users/:username/profile-stats", NewGetPublicProfileStatsHandler(&fakeGetPublicProfileStatsUseCase{err: tt.err}).Handle)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/users/ada/profile-stats", nil))
		if recorder.Code != tt.status {
			t.Fatalf("status = %d, want %d; body=%s", recorder.Code, tt.status, recorder.Body.String())
		}
	}
}
