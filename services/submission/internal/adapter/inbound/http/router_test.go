package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/response"
	"go-judge-system/services/submission/internal/adapter/inbound/http/handler"
	userhandler "go-judge-system/services/submission/internal/adapter/inbound/http/handler/user"
	"go-judge-system/services/submission/internal/application/dto"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type fakeRouterCreateSubmissionUseCase struct{}

func (*fakeRouterCreateSubmissionUseCase) Execute(
	context.Context,
	auth.Claims,
	dto.CreateSubmissionRequest,
) (dto.CreateSubmissionResponse, error) {
	return dto.CreateSubmissionResponse{}, nil
}

type fakeRouterGetSubmissionUseCase struct {
	calls int
}

func (f *fakeRouterGetSubmissionUseCase) Execute(
	context.Context,
	auth.Claims,
	dto.GetSubmissionRequest,
) (dto.GetSubmissionResponse, error) {
	f.calls++
	return dto.GetSubmissionResponse{}, nil
}

func TestRouterRegistersAuthenticatedSubmissionRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	getUseCase := &fakeRouterGetSubmissionUseCase{}
	userHandler := handler.NewUserHandler(
		userhandler.NewCreateSubmissionHandler(&fakeRouterCreateSubmissionUseCase{}),
		userhandler.NewGetSubmissionHandler(getUseCase),
	)

	authCalls := 0
	authMiddleware := func(c *gin.Context) {
		authCalls++
		response.Error(c, response.CodeUnauthorized, "unauthorized")
	}

	router := NewRouter(userHandler, authMiddleware, zap.NewNop())
	router.SetupRoutes()

	routeCounts := map[string]int{}
	for _, route := range router.engine.Routes() {
		routeCounts[route.Method+" "+route.Path]++
	}
	if routeCounts[http.MethodPost+" /api/v1/submissions"] != 1 {
		t.Fatalf("POST route count = %d, want 1", routeCounts[http.MethodPost+" /api/v1/submissions"])
	}
	if routeCounts[http.MethodGet+" /api/v1/submissions/:submission_id"] != 1 {
		t.Fatalf(
			"GET route count = %d, want 1",
			routeCounts[http.MethodGet+" /api/v1/submissions/:submission_id"],
		)
	}
	for _, staleRoute := range []string{
		http.MethodGet + " /api/v1/submissions/:id",
		http.MethodGet + " /api/v1/my/submissions/:id",
		http.MethodGet + " /api/v1/admin/submissions/:id",
	} {
		if routeCounts[staleRoute] != 0 {
			t.Fatalf("stale route %q count = %d, want 0", staleRoute, routeCounts[staleRoute])
		}
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/submissions/77", nil)
	router.engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
	}
	if authCalls != 1 {
		t.Fatalf("auth middleware calls = %d, want 1", authCalls)
	}
	if getUseCase.calls != 0 {
		t.Fatalf("get use case calls = %d, want 0", getUseCase.calls)
	}
}
