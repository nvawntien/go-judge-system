package http

import (
	"context"
	"encoding/json"
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

type fakeRouterListMySubmissionsUseCase struct {
	response dto.ListMySubmissionsResponse
	claims   auth.Claims
	req      dto.ListMySubmissionsRequest
	calls    int
}

func (f *fakeRouterListMySubmissionsUseCase) Execute(
	_ context.Context,
	claims auth.Claims,
	req dto.ListMySubmissionsRequest,
) (dto.ListMySubmissionsResponse, error) {
	f.calls++
	f.claims = claims
	f.req = req
	return f.response, nil
}

func TestRouterRegistersAuthenticatedSubmissionRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	getUseCase := &fakeRouterGetSubmissionUseCase{}
	listUseCase := &fakeRouterListMySubmissionsUseCase{}
	userHandler := handler.NewUserHandler(
		userhandler.NewCreateSubmissionHandler(&fakeRouterCreateSubmissionUseCase{}),
		userhandler.NewGetSubmissionHandler(getUseCase),
		userhandler.NewListMySubmissionsHandler(listUseCase),
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
	if routeCounts[http.MethodGet+" /api/v1/me/submissions"] != 1 {
		t.Fatalf(
			"GET my submissions route count = %d, want 1",
			routeCounts[http.MethodGet+" /api/v1/me/submissions"],
		)
	}
	for _, staleRoute := range []string{
		http.MethodGet + " /api/v1/submissions/:id",
		http.MethodGet + " /api/v1/my/submissions/:id",
		http.MethodGet + " /api/v1/admin/submissions/:id",
		http.MethodGet + " /api/v1/my/submissions",
	} {
		if routeCounts[staleRoute] != 0 {
			t.Fatalf("stale route %q count = %d, want 0", staleRoute, routeCounts[staleRoute])
		}
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/submissions", nil)
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
	if listUseCase.calls != 0 {
		t.Fatalf("list use case calls = %d, want 0", listUseCase.calls)
	}
}

func TestRouterListMySubmissionsUsesGenericHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	listUseCase := &fakeRouterListMySubmissionsUseCase{
		response: dto.ListMySubmissionsResponse{
			Items: []dto.SubmissionListItem{},
			Pagination: dto.PaginationResponse{
				Page: 2, Limit: 10,
			},
		},
	}
	userHandler := handler.NewUserHandler(
		userhandler.NewCreateSubmissionHandler(&fakeRouterCreateSubmissionUseCase{}),
		userhandler.NewGetSubmissionHandler(&fakeRouterGetSubmissionUseCase{}),
		userhandler.NewListMySubmissionsHandler(listUseCase),
	)

	claims := auth.Claims{
		UserID:   "verified-actor",
		Username: "verified-name",
		Role:     "admin",
	}
	authMiddleware := func(c *gin.Context) {
		auth.SetClaims(c, claims)
		c.Next()
	}

	router := NewRouter(userHandler, authMiddleware, zap.NewNop())
	router.SetupRoutes()

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/me/submissions?page=2&limit=10&status=PENDING"+
			"&language=GO&problem_id=42&user_id=attacker",
		nil,
	)
	router.engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if listUseCase.calls != 1 {
		t.Fatalf("list use case calls = %d, want 1", listUseCase.calls)
	}
	got := listUseCase.req
	if listUseCase.claims.UserID != claims.UserID || listUseCase.claims.Role != claims.Role {
		t.Fatalf("claims = %+v, want verified claims %+v", listUseCase.claims, claims)
	}
	if got.Page == nil || *got.Page != 2 ||
		got.Limit == nil || *got.Limit != 10 ||
		got.Status != "PENDING" ||
		got.Language != "GO" ||
		got.ProblemID == nil || *got.ProblemID != 42 {
		t.Fatalf("bound request = %+v", got)
	}

	var envelope struct {
		Status string                        `json:"status"`
		Code   int                           `json:"code"`
		Data   dto.ListMySubmissionsResponse `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Status != "success" ||
		envelope.Code != response.CodeSuccess ||
		envelope.Data.Items == nil {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
}
