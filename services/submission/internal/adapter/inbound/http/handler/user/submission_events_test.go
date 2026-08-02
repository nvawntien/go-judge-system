package user

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go-judge-system/pkg/config"
	"go-judge-system/pkg/response"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type fakeEventsSnapshotRepository struct {
	snapshot *entity.SubmissionStreamSnapshot
	err      error
	calls    int
	id       int64
	calledCh chan struct{}
}

func (r *fakeEventsSnapshotRepository) GetStreamSnapshot(
	ctx context.Context,
	submissionID int64,
) (*entity.SubmissionStreamSnapshot, error) {
	r.calls++
	r.id = submissionID
	if r.calledCh != nil && r.calls == 1 {
		close(r.calledCh)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return r.snapshot, r.err
}

type fakeEventsTicketService struct {
	claims entity.SubmissionStreamTicketClaims
	err    error
	raw    string
	calls  int
}

func (*fakeEventsTicketService) Issue(string, int64) (string, time.Time, error) {
	return "", time.Time{}, nil
}

func (s *fakeEventsTicketService) Verify(raw string) (entity.SubmissionStreamTicketClaims, error) {
	s.calls++
	s.raw = raw
	return s.claims, s.err
}

type fakeEventsHub struct {
	events           chan entity.SubmissionEvent
	submissionID     int64
	subscribeCalls   int
	unsubscribeCalls atomic.Int32
}

func newFakeEventsHub() *fakeEventsHub {
	return &fakeEventsHub{events: make(chan entity.SubmissionEvent, 2)}
}

func (h *fakeEventsHub) Subscribe(submissionID int64) (<-chan entity.SubmissionEvent, func()) {
	h.subscribeCalls++
	h.submissionID = submissionID
	return h.events, func() {
		h.unsubscribeCalls.Add(1)
	}
}

func (h *fakeEventsHub) Publish(event entity.SubmissionEvent) {
	h.events <- event
}

func performSubmissionEventsRequest(
	t *testing.T,
	path string,
	reqCtx context.Context,
	repo *fakeEventsSnapshotRepository,
	ticketService *fakeEventsTicketService,
	hub *fakeEventsHub,
	cfg config.SSEConfig,
) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = time.Hour
	}
	handler := NewSubmissionEventsHandler(repo, ticketService, hub, cfg, zap.NewNop())
	router := gin.New()
	router.GET("/events/submissions/:submission_id", handler.Handle)

	recorder := httptest.NewRecorder()
	if reqCtx == nil {
		reqCtx = context.Background()
	}
	req := httptest.NewRequest(http.MethodGet, path, nil).WithContext(reqCtx)
	req.Header.Set("Origin", "http://localhost:3000")
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestSubmissionEventsHandlerRejectsInvalidTicketBeforeSnapshot(t *testing.T) {
	repo := &fakeEventsSnapshotRepository{}
	ticketService := &fakeEventsTicketService{err: domain.ErrInvalidStreamTicket}
	hub := newFakeEventsHub()

	recorder := performSubmissionEventsRequest(
		t,
		"/events/submissions/77?ticket=opaque-secret-ticket",
		nil,
		repo,
		ticketService,
		hub,
		config.SSEConfig{AllowedOrigin: "http://localhost:3000"},
	)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", recorder.Code, recorder.Body.String())
	}
	if repo.calls != 0 || hub.subscribeCalls != 0 {
		t.Fatalf("repo/hub calls = %d/%d, want 0/0", repo.calls, hub.subscribeCalls)
	}
	if strings.Contains(recorder.Body.String(), "opaque-secret-ticket") {
		t.Fatalf("response must not echo raw ticket: %s", recorder.Body.String())
	}
}

func TestSubmissionEventsHandlerRejectsWrongSubmissionOrOwner(t *testing.T) {
	tests := []struct {
		name       string
		claims     entity.SubmissionStreamTicketClaims
		snapshot   *entity.SubmissionStreamSnapshot
		wantStatus int
	}{
		{
			name:       "ticket submission mismatch",
			claims:     entity.SubmissionStreamTicketClaims{Purpose: entity.SubmissionStreamTicketPurpose, UserID: "owner", SubmissionID: 88},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "ticket user differs from owner",
			claims:     entity.SubmissionStreamTicketClaims{Purpose: entity.SubmissionStreamTicketPurpose, UserID: "attacker", SubmissionID: 77},
			snapshot:   &entity.SubmissionStreamSnapshot{SubmissionID: 77, UserID: "owner", AttemptID: "attempt-1", Status: entity.StatusJudging},
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeEventsSnapshotRepository{snapshot: tt.snapshot}
			ticketService := &fakeEventsTicketService{claims: tt.claims}
			hub := newFakeEventsHub()
			recorder := performSubmissionEventsRequest(
				t,
				"/events/submissions/77?ticket=opaque",
				nil,
				repo,
				ticketService,
				hub,
				config.SSEConfig{AllowedOrigin: "http://localhost:3000"},
			)
			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
		})
	}
}

func TestSubmissionEventsHandlerSendsTerminalSnapshotAndCloses(t *testing.T) {
	updatedAt := time.Date(2026, 7, 30, 9, 0, 0, 0, time.UTC)
	repo := &fakeEventsSnapshotRepository{
		snapshot: &entity.SubmissionStreamSnapshot{
			SubmissionID: 77,
			UserID:       "owner",
			AttemptID:    "attempt-1",
			Status:       entity.StatusAccepted,
			UpdatedAt:    updatedAt,
		},
	}
	ticketService := &fakeEventsTicketService{
		claims: entity.SubmissionStreamTicketClaims{Purpose: entity.SubmissionStreamTicketPurpose, UserID: "owner", SubmissionID: 77},
	}
	hub := newFakeEventsHub()

	recorder := performSubmissionEventsRequest(
		t,
		"/events/submissions/77?ticket=opaque",
		nil,
		repo,
		ticketService,
		hub,
		config.SSEConfig{AllowedOrigin: "http://localhost:3000"},
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	for name, want := range map[string]string{
		"Content-Type":                "text/event-stream",
		"Cache-Control":               "no-cache, no-transform",
		"Connection":                  "keep-alive",
		"X-Accel-Buffering":           "no",
		"Access-Control-Allow-Origin": "http://localhost:3000",
		"Vary":                        "Origin",
	} {
		if got := recorder.Header().Get(name); got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}
	body := recorder.Body.String()
	for _, want := range []string{
		"retry: 3000",
		"event: submission.completed",
		`"submission_id":77`,
		`"attempt_id":"attempt-1"`,
		`"status":"ACCEPTED"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q: %s", want, body)
		}
	}
	for _, forbidden := range []string{"source_code", "compile_output", "error_message", "expected_output", "object_key"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("body leaks %q: %s", forbidden, body)
		}
	}
	if hub.unsubscribeCalls.Load() != 1 {
		t.Fatalf("unsubscribe calls = %d, want 1", hub.unsubscribeCalls.Load())
	}
}

func TestSubmissionEventsHandlerHeartbeatAndContextCancelCleanup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	repo := &fakeEventsSnapshotRepository{
		snapshot: &entity.SubmissionStreamSnapshot{
			SubmissionID: 77,
			UserID:       "owner",
			AttemptID:    "attempt-1",
			Status:       entity.StatusJudging,
			UpdatedAt:    time.Date(2026, 7, 30, 9, 0, 0, 0, time.UTC),
		},
	}
	ticketService := &fakeEventsTicketService{
		claims: entity.SubmissionStreamTicketClaims{Purpose: entity.SubmissionStreamTicketPurpose, UserID: "owner", SubmissionID: 77},
	}
	hub := newFakeEventsHub()

	recorder := performSubmissionEventsRequest(
		t,
		"/events/submissions/77?ticket=opaque",
		ctx,
		repo,
		ticketService,
		hub,
		config.SSEConfig{AllowedOrigin: "http://localhost:3000", HeartbeatInterval: 5 * time.Millisecond},
	)

	body := recorder.Body.String()
	if !strings.Contains(body, "event: submission.snapshot") || !strings.Contains(body, ": heartbeat") {
		t.Fatalf("body missing snapshot or heartbeat: %s", body)
	}
	if hub.unsubscribeCalls.Load() != 1 {
		t.Fatalf("unsubscribe calls = %d, want 1", hub.unsubscribeCalls.Load())
	}
}

func TestSubmissionEventsHandlerSendsHubTerminalEventAndIgnoresAttemptMismatch(t *testing.T) {
	repo := &fakeEventsSnapshotRepository{
		snapshot: &entity.SubmissionStreamSnapshot{
			SubmissionID: 77,
			UserID:       "owner",
			AttemptID:    "attempt-new",
			Status:       entity.StatusJudging,
			UpdatedAt:    time.Date(2026, 7, 30, 9, 0, 0, 0, time.UTC),
		},
	}
	ticketService := &fakeEventsTicketService{
		claims: entity.SubmissionStreamTicketClaims{Purpose: entity.SubmissionStreamTicketPurpose, UserID: "owner", SubmissionID: 77},
	}
	hub := newFakeEventsHub()
	hub.Publish(entity.SubmissionEvent{SubmissionID: 77, AttemptID: "attempt-old", Status: "ACCEPTED"})
	hub.Publish(entity.SubmissionEvent{
		SubmissionID: 77,
		AttemptID:    "attempt-new",
		Status:       "RUNTIME_ERROR",
		UpdatedAt:    time.Date(2026, 7, 30, 9, 1, 0, 0, time.UTC),
	})

	recorder := performSubmissionEventsRequest(
		t,
		"/events/submissions/77?ticket=opaque",
		nil,
		repo,
		ticketService,
		hub,
		config.SSEConfig{AllowedOrigin: "http://localhost:3000"},
	)

	body := recorder.Body.String()
	if strings.Contains(body, "attempt-old") {
		t.Fatalf("old attempt leaked into stream: %s", body)
	}
	if !strings.Contains(body, "event: submission.completed") ||
		!strings.Contains(body, `"status":"RUNTIME_ERROR"`) {
		t.Fatalf("body missing terminal event: %s", body)
	}
	if hub.unsubscribeCalls.Load() != 1 {
		t.Fatalf("unsubscribe calls = %d, want 1", hub.unsubscribeCalls.Load())
	}
}

func TestSubmissionEventsHandlerMapsSnapshotRepositoryErrors(t *testing.T) {
	repo := &fakeEventsSnapshotRepository{err: errors.New("database unavailable")}
	ticketService := &fakeEventsTicketService{
		claims: entity.SubmissionStreamTicketClaims{Purpose: entity.SubmissionStreamTicketPurpose, UserID: "owner", SubmissionID: 77},
	}
	hub := newFakeEventsHub()

	recorder := performSubmissionEventsRequest(
		t,
		"/events/submissions/77?ticket=opaque",
		nil,
		repo,
		ticketService,
		hub,
		config.SSEConfig{AllowedOrigin: "http://localhost:3000"},
	)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", recorder.Code, recorder.Body.String())
	}
	if hub.unsubscribeCalls.Load() != 1 {
		t.Fatalf("unsubscribe calls = %d, want 1", hub.unsubscribeCalls.Load())
	}
	var envelope response.APIResponse
	if err := jsonUnmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Code != response.CodeInternalServer {
		t.Fatalf("code = %d, want %d", envelope.Code, response.CodeInternalServer)
	}
}

func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
