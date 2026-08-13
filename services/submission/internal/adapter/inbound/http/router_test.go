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

type fakeRouterGetMyProfileStatsUseCase struct {
	response dto.GetMyProfileStatsResponse
	claims   auth.Claims
	calls    int
}

type fakeRouterGetPublicProfileStatsUseCase struct {
	response dto.GetMyProfileStatsResponse
	req      dto.GetPublicProfileStatsRequest
	calls    int
}

func (f *fakeRouterGetPublicProfileStatsUseCase) Execute(
	_ context.Context,
	req dto.GetPublicProfileStatsRequest,
) (dto.GetMyProfileStatsResponse, error) {
	f.calls++
	f.req = req
	return f.response, nil
}

func (f *fakeRouterGetMyProfileStatsUseCase) Execute(
	_ context.Context,
	claims auth.Claims,
) (dto.GetMyProfileStatsResponse, error) {
	f.calls++
	f.claims = claims
	return f.response, nil
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

type fakeRouterRejudgeSubmissionUseCase struct {
	response dto.RejudgeAdminSubmissionResponse
	claims   auth.Claims
	req      dto.RejudgeAdminSubmissionRequest
	calls    int
}

func (f *fakeRouterRejudgeSubmissionUseCase) Execute(
	_ context.Context,
	claims auth.Claims,
	req dto.RejudgeAdminSubmissionRequest,
) (dto.RejudgeAdminSubmissionResponse, error) {
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
	profileStatsUseCase *fakeRouterGetMyProfileStatsUseCase,
	publicProfileStatsUseCase *fakeRouterGetPublicProfileStatsUseCase,
) *handler.UserHandler {
	return handler.NewUserHandler(
		userhandler.NewCreateSubmissionHandler(createUseCase),
		userhandler.NewRunCodeHandler(runUseCase),
		userhandler.NewGetSubmissionHandler(getUseCase),
		userhandler.NewListMySubmissionsHandler(listUseCase),
		userhandler.NewGetMyProfileStatsHandler(profileStatsUseCase),
		userhandler.NewGetPublicProfileStatsHandler(publicProfileStatsUseCase),
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
	rejudgeUseCase := &fakeRouterRejudgeSubmissionUseCase{}
	userHandler := newTestUserHandler(
		&fakeRouterCreateSubmissionUseCase{},
		&fakeRouterRunCodeUseCase{},
		getUseCase,
		listUseCase,
		&fakeRouterGetMyProfileStatsUseCase{},
		&fakeRouterGetPublicProfileStatsUseCase{},
	)
	adminHandler := handler.NewAdminHandler(
		adminhandler.NewListSubmissionsHandler(adminListUseCase),
		adminhandler.NewGetSubmissionDetailHandler(adminDetailUseCase),
		adminhandler.NewRejudgeSubmissionHandler(rejudgeUseCase),
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
	if routeCounts[http.MethodGet+" /api/v1/me/profile-stats"] != 1 {
		t.Fatalf(
			"GET profile stats route count = %d, want 1",
			routeCounts[http.MethodGet+" /api/v1/me/profile-stats"],
		)
	}
	if routeCounts[http.MethodGet+" /api/v1/users/:username/profile-stats"] != 1 {
		t.Fatalf(
			"GET public profile stats route count = %d, want 1",
			routeCounts[http.MethodGet+" /api/v1/users/:username/profile-stats"],
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
	if routeCounts[http.MethodPost+" /api/v1/admin/submissions/:submission_id/rejudge"] != 1 {
		t.Fatalf(
			"POST admin rejudge route count = %d, want 1",
			routeCounts[http.MethodPost+" /api/v1/admin/submissions/:submission_id/rejudge"],
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
		http.MethodPost + " /api/v1/admin/submissions/:id/rejudge",
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
	if rejudgeUseCase.calls != 0 {
		t.Fatalf("rejudge use case calls = %d, want 0", rejudgeUseCase.calls)
	}
}

func TestRouterServesPublicProfileStatsWithoutGatewayClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	publicStatsUseCase := &fakeRouterGetPublicProfileStatsUseCase{
		response: dto.GetMyProfileStatsResponse{SolvedProblems: 2},
	}
	userHandler := newTestUserHandler(
		&fakeRouterCreateSubmissionUseCase{},
		&fakeRouterRunCodeUseCase{},
		&fakeRouterGetSubmissionUseCase{},
		&fakeRouterListMySubmissionsUseCase{},
		&fakeRouterGetMyProfileStatsUseCase{},
		publicStatsUseCase,
	)
	adminHandler := handler.NewAdminHandler(
		adminhandler.NewListSubmissionsHandler(&fakeRouterListAdminSubmissionsUseCase{}),
		adminhandler.NewGetSubmissionDetailHandler(&fakeRouterGetAdminSubmissionDetailUseCase{}),
	)
	authCalls := 0
	router := NewRouter(userHandler, adminHandler, func(c *gin.Context) {
		authCalls++
		response.Error(c, response.CodeUnauthorized, "unauthorized")
	}, zap.NewNop())
	router.SetupRoutes()

	recorder := httptest.NewRecorder()
	router.engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/users/ada/profile-stats", nil))
	if recorder.Code != http.StatusOK || authCalls != 0 || publicStatsUseCase.calls != 1 || publicStatsUseCase.req.Username != "ada" {
		t.Fatalf("status/auth/usecase/request = %d/%d/%d/%+v", recorder.Code, authCalls, publicStatsUseCase.calls, publicStatsUseCase.req)
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
		&fakeRouterGetMyProfileStatsUseCase{},
		&fakeRouterGetPublicProfileStatsUseCase{},
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
		&fakeRouterGetMyProfileStatsUseCase{},
		&fakeRouterGetPublicProfileStatsUseCase{},
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
		&fakeRouterGetMyProfileStatsUseCase{},
		&fakeRouterGetPublicProfileStatsUseCase{},
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

func TestRouterRejudgeSubmissionUsesGenericHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rejudgeUseCase := &fakeRouterRejudgeSubmissionUseCase{
		response: dto.RejudgeAdminSubmissionResponse{
			SubmissionID:   77,
			AttemptID:      "attempt-rejudge",
			Status:         "PENDING",
			AttemptTrigger: "ADMIN_REJUDGE",
			EnqueuedAt:     time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC),
		},
	}
	userHandler := newTestUserHandler(
		&fakeRouterCreateSubmissionUseCase{},
		&fakeRouterRunCodeUseCase{},
		&fakeRouterGetSubmissionUseCase{},
		&fakeRouterListMySubmissionsUseCase{},
		&fakeRouterGetMyProfileStatsUseCase{},
		&fakeRouterGetPublicProfileStatsUseCase{},
	)
	adminHandler := handler.NewAdminHandler(
		adminhandler.NewListSubmissionsHandler(&fakeRouterListAdminSubmissionsUseCase{}),
		adminhandler.NewGetSubmissionDetailHandler(&fakeRouterGetAdminSubmissionDetailUseCase{}),
		adminhandler.NewRejudgeSubmissionHandler(rejudgeUseCase),
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
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/submissions/77/rejudge", nil)
	router.engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if rejudgeUseCase.calls != 1 {
		t.Fatalf("rejudge use case calls = %d, want 1", rejudgeUseCase.calls)
	}
	if rejudgeUseCase.claims.UserID != claims.UserID || rejudgeUseCase.claims.Role != claims.Role {
		t.Fatalf("claims = %+v, want verified claims %+v", rejudgeUseCase.claims, claims)
	}
	if rejudgeUseCase.req.SubmissionID != 77 {
		t.Fatalf("submission ID = %d, want 77", rejudgeUseCase.req.SubmissionID)
	}
}
