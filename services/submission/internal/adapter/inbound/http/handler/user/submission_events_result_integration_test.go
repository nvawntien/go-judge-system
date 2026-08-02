package user

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-judge-system/pkg/config"
	pkgjudge "go-judge-system/pkg/judge"
	streamadapter "go-judge-system/services/submission/internal/adapter/outbound/stream"
	"go-judge-system/services/submission/internal/application/port/outbound"
	resultusecase "go-judge-system/services/submission/internal/application/usecase/result"
	"go-judge-system/services/submission/internal/domain/entity"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type integrationTxManager struct{}

func (integrationTxManager) ExecuteInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type integrationSubmissionRepo struct {
	submission *entity.Submission
	updates    int
}

func (r *integrationSubmissionRepo) Create(context.Context, *entity.Submission) error { return nil }
func (r *integrationSubmissionRepo) GetByID(context.Context, int64) (*entity.Submission, error) {
	return nil, nil
}
func (r *integrationSubmissionRepo) GetByIDForUpdate(context.Context, int64) (*entity.Submission, error) {
	copy := *r.submission
	return &copy, nil
}
func (r *integrationSubmissionRepo) Update(_ context.Context, submission *entity.Submission) error {
	copy := *submission
	r.submission = &copy
	r.updates++
	return nil
}
func (r *integrationSubmissionRepo) List(context.Context, outbound.ListSubmissionsFilter) (outbound.ListSubmissionsResult, error) {
	return outbound.ListSubmissionsResult{}, nil
}
func (r *integrationSubmissionRepo) ResultSummaries(
	context.Context,
	[]int64,
) (map[int64]outbound.SubmissionResultSummary, error) {
	return map[int64]outbound.SubmissionResultSummary{}, nil
}

type integrationSubmissionResultRepo struct {
	calls int
}

func (r *integrationSubmissionResultRepo) GetBySubmissionID(context.Context, int64) ([]*entity.SubmissionResult, error) {
	return nil, nil
}
func (r *integrationSubmissionResultRepo) DeleteBySubmissionID(context.Context, int64) error {
	return nil
}
func (r *integrationSubmissionResultRepo) ReplaceBySubmissionIDAndAttemptID(
	context.Context,
	int64,
	string,
	[]*entity.SubmissionResult,
) error {
	r.calls++
	return nil
}

func TestSubmissionEventsHandlerReceivesApplyJudgeResultCommittedEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	snapshotRead := make(chan struct{})
	updatedAt := time.Date(2026, 8, 2, 9, 0, 0, 0, time.UTC)
	snapshotRepo := &fakeEventsSnapshotRepository{
		snapshot: &entity.SubmissionStreamSnapshot{
			SubmissionID: 77,
			UserID:       "owner",
			AttemptID:    "attempt-77",
			Status:       entity.StatusJudging,
			UpdatedAt:    updatedAt,
		},
		calledCh: snapshotRead,
	}
	ticketService := &fakeEventsTicketService{
		claims: entity.SubmissionStreamTicketClaims{
			Purpose:      entity.SubmissionStreamTicketPurpose,
			UserID:       "owner",
			SubmissionID: 77,
		},
	}
	eventHub := streamadapter.NewSubmissionEventHub()
	handler := NewSubmissionEventsHandler(
		snapshotRepo,
		ticketService,
		eventHub,
		config.SSEConfig{AllowedOrigin: "http://localhost:3000", HeartbeatInterval: time.Hour},
		zap.NewNop(),
	)
	router := gin.New()
	router.GET("/events/submissions/:submission_id", handler.Handle)

	requestCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events/submissions/77?ticket=opaque", nil).WithContext(requestCtx)
	req.Header.Set("Origin", "http://localhost:3000")

	done := make(chan struct{})
	go func() {
		router.ServeHTTP(recorder, req)
		close(done)
	}()

	select {
	case <-snapshotRead:
	case <-done:
		t.Fatalf("SSE request ended before subscribing; status=%d body=%s", recorder.Code, recorder.Body.String())
	case <-time.After(time.Second):
		t.Fatal("SSE handler did not read snapshot")
	}

	submissionRepo := &integrationSubmissionRepo{
		submission: &entity.Submission{
			ID:               77,
			UserID:           "owner",
			CurrentAttemptID: "attempt-77",
			Status:           entity.StatusJudging,
			UpdatedAt:        updatedAt,
		},
	}
	resultRepo := &integrationSubmissionResultRepo{}
	uc := resultusecase.NewApplyJudgeResultUseCase(
		submissionRepo,
		resultRepo,
		integrationTxManager{},
		eventHub,
		zap.NewNop(),
	)
	if err := uc.Execute(context.Background(), pkgjudge.ResultMessage{
		SubmissionID: 77,
		AttemptID:    "attempt-77",
		Status:       string(entity.StatusAccepted),
		TestCases: []pkgjudge.TestCaseResultItem{{
			Index:  1,
			Status: string(entity.ResultAccepted),
		}},
	}); err != nil {
		t.Fatalf("ApplyJudgeResult Execute() error = %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SSE stream did not close after terminal event")
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if submissionRepo.updates != 1 || resultRepo.calls != 1 {
		t.Fatalf("persistence calls = submission updates %d result replacements %d, want 1/1", submissionRepo.updates, resultRepo.calls)
	}
	if submissionRepo.submission.Status != entity.StatusAccepted {
		t.Fatalf("committed status = %s, want ACCEPTED", submissionRepo.submission.Status)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		"event: submission.snapshot",
		"event: submission.completed",
		`"submission_id":77`,
		`"attempt_id":"attempt-77"`,
		`"status":"ACCEPTED"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("SSE body missing %q: %s", want, body)
		}
	}
	for _, forbidden := range []string{"source_code", "compile_output", "error_message", "expected_output", "actual_output"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("SSE event leaks %q: %s", forbidden, body)
		}
	}
}
