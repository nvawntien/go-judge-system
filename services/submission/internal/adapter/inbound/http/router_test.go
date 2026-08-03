package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/config"
	"go-judge-system/pkg/response"
	"go-judge-system/services/submission/internal/adapter/inbound/http/handler"
	adminhandler "go-judge-system/services/submission/internal/adapter/inbound/http/handler/admin"
	userhandler "go-judge-system/services/submission/internal/adapter/inbound/http/handler/user"
	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/domain/entity"

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

type fakeRouterRunCodeUseCase struct{}

func (*fakeRouterRunCodeUseCase) Execute(
	context.Context,
	auth.Claims,
	dto.RunCodeRequest,
) (dto.RunCodeResponse, error) {
	return dto.RunCodeResponse{}, nil
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

type fakeRouterListAdminSubmissionsUseCase struct {
	response dto.ListAdminSubmissionsResponse
	claims   auth.Claims
	req      dto.ListAdminSubmissionsRequest
	calls    int
}

func (f *fakeRouterListAdminSubmissionsUseCase) Execute(
	_ context.Context,
	claims auth.Claims,
	req dto.ListAdminSubmissionsRequest,
) (dto.ListAdminSubmissionsResponse, error) {
	f.calls++
	f.claims = claims
	f.req = req
	return f.response, nil
}

type fakeRouterGetAdminSubmissionDetailUseCase struct {
	response dto.GetAdminSubmissionDetailResponse
	claims   auth.Claims
	req      dto.GetAdminSubmissionDetailRequest
	calls    int
}

func (f *fakeRouterGetAdminSubmissionDetailUseCase) Execute(
	_ context.Context,
	claims auth.Claims,
	req dto.GetAdminSubmissionDetailRequest,
) (dto.GetAdminSubmissionDetailResponse, error) {
	f.calls++
	f.claims = claims
	f.req = req
	return f.response, nil
}

type fakeRouterIssueStreamTicketUseCase struct{}

func (*fakeRouterIssueStreamTicketUseCase) Execute(
	context.Context,
	auth.Claims,
	dto.IssueSubmissionStreamTicketRequest,
) (dto.IssueSubmissionStreamTicketResponse, error) {
	return dto.IssueSubmissionStreamTicketResponse{}, nil
}

type fakeRouterSnapshotRepository struct{}

func (*fakeRouterSnapshotRepository) GetStreamSnapshot(
	context.Context,
	int64,
) (*entity.SubmissionStreamSnapshot, error) {
	return nil, nil
}

type fakeRouterStreamTicketService struct{}

func (*fakeRouterStreamTicketService) Issue(string, int64) (string, time.Time, error) {
	return "", time.Time{}, nil
}

func (*fakeRouterStreamTicketService) Verify(string) (entity.SubmissionStreamTicketClaims, error) {
	return entity.SubmissionStreamTicketClaims{}, nil
}

type fakeRouterEventHub struct{}

func (*fakeRouterEventHub) Subscribe(int64) (<-chan entity.SubmissionEvent, func()) {
	return make(chan entity.SubmissionEvent), func() {}
}

func (*fakeRouterEventHub) Publish(entity.SubmissionEvent) {}

func newTestUserHandler(
	createUseCase *fakeRouterCreateSubmissionUseCase,
	runUseCase *fakeRouterRunCodeUseCase,
	getUseCase *fakeRouterGetSubmissionUseCase,
	listUseCase *fakeRouterListMySubmissionsUseCase,
) *handler.UserHandler {
	return handler.NewUserHandler(
		userhandler.NewCreateSubmissionHandler(createUseCase),
		userhandler.NewRunCodeHandler(runUseCase),
		userhandler.NewGetSubmissionHandler(getUseCase),
		userhandler.NewListMySubmissionsHandler(listUseCase),
		userhandler.NewIssueSubmissionStreamTicketHandler(&fakeRouterIssueStreamTicketUseCase{}),
		userhandler.NewSubmissionEventsHandler(
			&fakeRouterSnapshotRepository{},
			&fakeRouterStreamTicketService{},
			&fakeRouterEventHub{},
			config.SSEConfig{HeartbeatInterval: time.Second, AllowedOrigin: "http://localhost:3000"},
			zap.NewNop(),
		),
	)
}

func TestRouterRegistersAuthenticatedSubmissionRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	getUseCase := &fakeRouterGetSubmissionUseCase{}
	listUseCase := &fakeRouterListMySubmissionsUseCase{}
	adminListUseCase := &fakeRouterListAdminSubmissionsUseCase{}
	adminDetailUseCase := &fakeRouterGetAdminSubmissionDetailUseCase{}
	userHandler := newTestUserHandler(
		&fakeRouterCreateSubmissionUseCase{},
		&fakeRouterRunCodeUseCase{},
		getUseCase,
		listUseCase,
	)
	adminHandler := handler.NewAdminHandler(
		adminhandler.NewListSubmissionsHandler(adminListUseCase),
		adminhandler.NewGetSubmissionDetailHandler(adminDetailUseCase),
	)

	authCalls := 0
	authMiddleware := func(c *gin.Context) {
		authCalls++
		response.Error(c, response.CodeUnauthorized, "unauthorized")
	}

	router := NewRouter(userHandler, adminHandler, authMiddleware, zap.NewNop())
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
	if routeCounts[http.MethodGet+" /api/v1/admin/submissions"] != 1 {
		t.Fatalf(
			"GET admin submissions route count = %d, want 1",
			routeCounts[http.MethodGet+" /api/v1/admin/submissions"],
		)
	}
	if routeCounts[http.MethodGet+" /api/v1/admin/submissions/:submission_id"] != 1 {
		t.Fatalf(
			"GET admin submission detail route count = %d, want 1",
			routeCounts[http.MethodGet+" /api/v1/admin/submissions/:submission_id"],
		)
	}
	if routeCounts[http.MethodPost+" /api/v1/submissions/:submission_id/events/ticket"] != 1 {
		t.Fatalf(
			"POST stream ticket route count = %d, want 1",
			routeCounts[http.MethodPost+" /api/v1/submissions/:submission_id/events/ticket"],
		)
	}
	if routeCounts[http.MethodGet+" /events/submissions/:submission_id"] != 1 {
		t.Fatalf(
			"SSE route count = %d, want 1",
			routeCounts[http.MethodGet+" /events/submissions/:submission_id"],
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
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/submissions", nil)
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
	if adminListUseCase.calls != 0 {
		t.Fatalf("admin list use case calls = %d, want 0", adminListUseCase.calls)
	}
	if adminDetailUseCase.calls != 0 {
		t.Fatalf("admin detail use case calls = %d, want 0", adminDetailUseCase.calls)
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
	userHandler := newTestUserHandler(
		&fakeRouterCreateSubmissionUseCase{},
		&fakeRouterRunCodeUseCase{},
		&fakeRouterGetSubmissionUseCase{},
		listUseCase,
	)
	adminHandler := handler.NewAdminHandler(
		adminhandler.NewListSubmissionsHandler(&fakeRouterListAdminSubmissionsUseCase{}),
		adminhandler.NewGetSubmissionDetailHandler(&fakeRouterGetAdminSubmissionDetailUseCase{}),
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

	router := NewRouter(userHandler, adminHandler, authMiddleware, zap.NewNop())
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

func TestRouterListAdminSubmissionsUsesGenericHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adminListUseCase := &fakeRouterListAdminSubmissionsUseCase{
		response: dto.ListAdminSubmissionsResponse{
			Items: []dto.AdminSubmissionListItem{},
			Pagination: dto.PaginationResponse{
				Page: 2, Limit: 10,
			},
		},
	}
	userHandler := newTestUserHandler(
		&fakeRouterCreateSubmissionUseCase{},
		&fakeRouterRunCodeUseCase{},
		&fakeRouterGetSubmissionUseCase{},
		&fakeRouterListMySubmissionsUseCase{},
	)
	adminHandler := handler.NewAdminHandler(
		adminhandler.NewListSubmissionsHandler(adminListUseCase),
		adminhandler.NewGetSubmissionDetailHandler(&fakeRouterGetAdminSubmissionDetailUseCase{}),
	)

	claims := auth.Claims{
		UserID:   "verified-admin",
		Username: "admin-name",
		Role:     "admin",
	}
	authMiddleware := func(c *gin.Context) {
		auth.SetClaims(c, claims)
		c.Next()
	}

	router := NewRouter(userHandler, adminHandler, authMiddleware, zap.NewNop())
	router.SetupRoutes()

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/admin/submissions?page=2&limit=10&status=PENDING"+
			"&language=GO&problem_id=42&user_id=user-123",
		nil,
	)
	router.engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if adminListUseCase.calls != 1 {
		t.Fatalf("admin list use case calls = %d, want 1", adminListUseCase.calls)
	}
	got := adminListUseCase.req
	if adminListUseCase.claims.UserID != claims.UserID || adminListUseCase.claims.Role != claims.Role {
		t.Fatalf("claims = %+v, want verified claims %+v", adminListUseCase.claims, claims)
	}
	if got.Page == nil || *got.Page != 2 ||
		got.Limit == nil || *got.Limit != 10 ||
		got.Status == nil || *got.Status != "PENDING" ||
		got.Language == nil || *got.Language != "GO" ||
		got.ProblemID == nil || *got.ProblemID != 42 ||
		got.UserID == nil || *got.UserID != "user-123" {
		t.Fatalf("bound request = %+v", got)
	}

	var envelope struct {
		Status string                           `json:"status"`
		Code   int                              `json:"code"`
		Data   dto.ListAdminSubmissionsResponse `json:"data"`
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

func TestRouterGetAdminSubmissionDetailUsesGenericHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adminDetailUseCase := &fakeRouterGetAdminSubmissionDetailUseCase{
		response: dto.GetAdminSubmissionDetailResponse{
			ID:           77,
			ProblemID:    42,
			ProblemTitle: "Two Sum",
			UserID:       "user-123",
			Username:     "alice",
			Language:     "CPP",
			Status:       "ACCEPTED",
			TestResults:  []dto.AdminSubmissionTestResult{},
		},
	}
	userHandler := newTestUserHandler(
		&fakeRouterCreateSubmissionUseCase{},
		&fakeRouterRunCodeUseCase{},
		&fakeRouterGetSubmissionUseCase{},
		&fakeRouterListMySubmissionsUseCase{},
	)
	adminHandler := handler.NewAdminHandler(
		adminhandler.NewListSubmissionsHandler(&fakeRouterListAdminSubmissionsUseCase{}),
		adminhandler.NewGetSubmissionDetailHandler(adminDetailUseCase),
	)

	claims := auth.Claims{
		UserID:   "verified-moderator",
		Username: "moderator-name",
		Role:     "moderator",
	}
	authMiddleware := func(c *gin.Context) {
		auth.SetClaims(c, claims)
		c.Next()
	}

	router := NewRouter(userHandler, adminHandler, authMiddleware, zap.NewNop())
	router.SetupRoutes()

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/submissions/77", nil)
	router.engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if adminDetailUseCase.calls != 1 {
		t.Fatalf("admin detail use case calls = %d, want 1", adminDetailUseCase.calls)
	}
	if adminDetailUseCase.claims.UserID != claims.UserID || adminDetailUseCase.claims.Role != claims.Role {
		t.Fatalf("claims = %+v, want verified claims %+v", adminDetailUseCase.claims, claims)
	}
	if adminDetailUseCase.req.SubmissionID != 77 {
		t.Fatalf("submission ID = %d, want 77", adminDetailUseCase.req.SubmissionID)
	}
}
