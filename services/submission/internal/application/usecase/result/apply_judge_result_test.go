package result

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	pkgjudge "go-judge-system/pkg/judge"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"
)

type fakeTxManager struct {
	called    bool
	commitErr error
}

func (m *fakeTxManager) ExecuteInTx(ctx context.Context, fn func(context.Context) error) error {
	m.called = true
	if err := fn(ctx); err != nil {
		return err
	}
	return m.commitErr
}

type fakeSubmissionRepo struct {
	submission *entity.Submission
	getErr     error
	updateErr  error
	lockedID   int64
	updates    []*entity.Submission
}

func (r *fakeSubmissionRepo) Create(context.Context, *entity.Submission) error { return nil }
func (r *fakeSubmissionRepo) GetByID(context.Context, int64) (*entity.Submission, error) {
	return nil, nil
}
func (r *fakeSubmissionRepo) GetByIDForUpdate(ctx context.Context, id int64) (*entity.Submission, error) {
	r.lockedID = id
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.getErr != nil {
		return nil, r.getErr
	}
	copy := *r.submission
	return &copy, nil
}
func (r *fakeSubmissionRepo) Update(ctx context.Context, submission *entity.Submission) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.updateErr != nil {
		return r.updateErr
	}
	copy := *submission
	r.updates = append(r.updates, &copy)
	r.submission = &copy
	return nil
}
func (r *fakeSubmissionRepo) List(context.Context, outbound.ListSubmissionsFilter) (outbound.ListSubmissionsResult, error) {
	return outbound.ListSubmissionsResult{}, nil
}
func (r *fakeSubmissionRepo) ResultSummaries(
	context.Context,
	[]int64,
) (map[int64]outbound.SubmissionResultSummary, error) {
	return map[int64]outbound.SubmissionResultSummary{}, nil
}

type fakeSubmissionResultRepo struct {
	replaceErr error
	calls      int
	submission int64
	attemptID  string
	results    []*entity.SubmissionResult
}

func (r *fakeSubmissionResultRepo) GetBySubmissionID(context.Context, int64) ([]*entity.SubmissionResult, error) {
	return nil, nil
}
func (r *fakeSubmissionResultRepo) DeleteBySubmissionID(context.Context, int64) error { return nil }
func (r *fakeSubmissionResultRepo) ReplaceBySubmissionIDAndAttemptID(ctx context.Context, submissionID int64, attemptID string, results []*entity.SubmissionResult) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.replaceErr != nil {
		return r.replaceErr
	}
	r.calls++
	r.submission = submissionID
	r.attemptID = attemptID
	r.results = results
	return nil
}

type fakeSubmissionEventHub struct {
	events []entity.SubmissionEvent
}

func (h *fakeSubmissionEventHub) Subscribe(int64) (<-chan entity.SubmissionEvent, func()) {
	return make(chan entity.SubmissionEvent), func() {}
}

func (h *fakeSubmissionEventHub) Publish(event entity.SubmissionEvent) {
	h.events = append(h.events, event)
}

func matchingSubmission() *entity.Submission {
	return &entity.Submission{
		ID:               77,
		ProblemID:        42,
		ProblemName:      "Two Sum",
		UserID:           "user-1",
		Username:         "alice",
		Language:         entity.LanguageGo,
		SourceCode:       "package main",
		CurrentAttemptID: "attempt-77",
		Status:           entity.StatusJudging,
		CreatedAt:        time.Now().Add(-time.Minute),
		UpdatedAt:        time.Now().Add(-time.Minute),
	}
}

func TestApplyJudgeResultMatchingAttemptAppliesTerminalStatuses(t *testing.T) {
	for _, status := range []entity.Status{
		entity.StatusAccepted,
		entity.StatusWrongAnswer,
		entity.StatusTimeLimitExceed,
		entity.StatusMemoryLimitExceed,
		entity.StatusRuntimeError,
		entity.StatusCompilationError,
		entity.StatusSystemError,
	} {
		t.Run(string(status), func(t *testing.T) {
			subRepo := &fakeSubmissionRepo{submission: matchingSubmission()}
			resultRepo := &fakeSubmissionResultRepo{}
			eventHub := &fakeSubmissionEventHub{}
			uc := NewApplyJudgeResultUseCase(subRepo, resultRepo, &fakeTxManager{}, eventHub, nil)
			execTime := 12
			memory := 128

			err := uc.Execute(context.Background(), pkgjudge.ResultMessage{
				SubmissionID:  77,
				AttemptID:     "attempt-77",
				Status:        string(status),
				ExecutionTime: &execTime,
				MemoryUsed:    &memory,
				TestCases: []pkgjudge.TestCaseResultItem{{
					Index:         1,
					Status:        string(entity.ResultAccepted),
					ExecutionTime: &execTime,
					MemoryUsed:    &memory,
				}},
			})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if subRepo.lockedID != 77 {
				t.Fatalf("locked submission = %d, want 77", subRepo.lockedID)
			}
			if len(subRepo.updates) != 1 || subRepo.updates[0].Status != status {
				t.Fatalf("updates = %+v, want one %s update", subRepo.updates, status)
			}
			if resultRepo.calls != 1 || resultRepo.attemptID != "attempt-77" || len(resultRepo.results) != 1 {
				t.Fatalf("result replace = calls %d attempt %q len %d", resultRepo.calls, resultRepo.attemptID, len(resultRepo.results))
			}
			if resultRepo.results[0].AttemptID != "attempt-77" {
				t.Fatalf("result attempt = %q, want attempt-77", resultRepo.results[0].AttemptID)
			}
			if len(eventHub.events) != 1 {
				t.Fatalf("published events = %d, want 1", len(eventHub.events))
			}
			event := eventHub.events[0]
			if event.SubmissionID != 77 || event.AttemptID != "attempt-77" || event.Status != string(status) {
				t.Fatalf("event = %+v, want committed submission status", event)
			}
			if !event.UpdatedAt.Equal(subRepo.updates[0].UpdatedAt) {
				t.Fatalf("event updated_at = %s, want committed %s", event.UpdatedAt, subRepo.updates[0].UpdatedAt)
			}
		})
	}
}

func TestApplyJudgeResultMapsOutputFieldsByVerdict(t *testing.T) {
	compileOutput := "main.go:8:2: undefined: value"
	runtimeError := "panic: runtime error: index out of range"
	unsafeSystemMessage := "/tmp/judge/internal stack"
	longRuntimeError := "/tmp/judge/run/main.go:7 " + strings.Repeat("x", maxStoredErrorMessageBytes+4)

	tests := []struct {
		name              string
		msg               pkgjudge.ResultMessage
		wantCompileOutput *string
		wantErrorMessage  *string
	}{
		{
			name: "compilation error keeps compile output only",
			msg: pkgjudge.ResultMessage{
				Status:        string(entity.StatusCompilationError),
				CompileOutput: &compileOutput,
				ErrorMessage:  &runtimeError,
			},
			wantCompileOutput: &compileOutput,
		},
		{
			name: "runtime error stores sanitized error message only",
			msg: pkgjudge.ResultMessage{
				Status:        string(entity.StatusRuntimeError),
				CompileOutput: &compileOutput,
				ErrorMessage:  &runtimeError,
			},
			wantErrorMessage: &runtimeError,
		},
		{
			name: "accepted clears stale outputs",
			msg:  pkgjudge.ResultMessage{Status: string(entity.StatusAccepted), CompileOutput: &compileOutput, ErrorMessage: &runtimeError},
		},
		{
			name: "wrong answer clears stale outputs",
			msg:  pkgjudge.ResultMessage{Status: string(entity.StatusWrongAnswer), CompileOutput: &compileOutput, ErrorMessage: &runtimeError},
		},
		{
			name:             "time limit gets public default message",
			msg:              pkgjudge.ResultMessage{Status: string(entity.StatusTimeLimitExceed)},
			wantErrorMessage: stringPointer(publicTimeLimitMessage),
		},
		{
			name:             "memory limit gets public default message",
			msg:              pkgjudge.ResultMessage{Status: string(entity.StatusMemoryLimitExceed)},
			wantErrorMessage: stringPointer(publicMemoryLimitMessage),
		},
		{
			name:             "system error ignores unsafe incoming message",
			msg:              pkgjudge.ResultMessage{Status: string(entity.StatusSystemError), ErrorMessage: &unsafeSystemMessage},
			wantErrorMessage: stringPointer(publicSystemErrorMessage),
		},
		{
			name: "runtime error sanitizes and truncates incoming message",
			msg:  pkgjudge.ResultMessage{Status: string(entity.StatusRuntimeError), ErrorMessage: &longRuntimeError},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			submission := matchingSubmission()
			staleCompile := "old compile"
			staleError := "old error"
			submission.CompileOutput = &staleCompile
			submission.ErrorMessage = &staleError
			subRepo := &fakeSubmissionRepo{submission: submission}
			resultRepo := &fakeSubmissionResultRepo{}
			uc := NewApplyJudgeResultUseCase(subRepo, resultRepo, &fakeTxManager{}, nil, nil)

			tt.msg.SubmissionID = 77
			tt.msg.AttemptID = "attempt-77"
			err := uc.Execute(context.Background(), tt.msg)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if len(subRepo.updates) != 1 {
				t.Fatalf("updates = %d, want 1", len(subRepo.updates))
			}
			got := subRepo.updates[0]
			if !sameStringPtr(got.CompileOutput, tt.wantCompileOutput) {
				t.Fatalf("compile_output = %v, want %v", got.CompileOutput, tt.wantCompileOutput)
			}
			if tt.name == "runtime error sanitizes and truncates incoming message" {
				if got.ErrorMessage == nil {
					t.Fatal("error_message = nil, want truncated sanitized message")
				}
				if len(*got.ErrorMessage) > maxStoredErrorMessageBytes+len("…") {
					t.Fatalf("error_message length = %d, want truncated", len(*got.ErrorMessage))
				}
				if strings.Contains(*got.ErrorMessage, "/tmp/") {
					t.Fatalf("error_message leaked internal path: %q", *got.ErrorMessage)
				}
				return
			}
			if !sameStringPtr(got.ErrorMessage, tt.wantErrorMessage) {
				t.Fatalf("error_message = %v, want %v", got.ErrorMessage, tt.wantErrorMessage)
			}
		})
	}
}

func TestApplyJudgeResultStaleOrLegacyAttemptIsAckNoop(t *testing.T) {
	for _, tt := range []struct {
		name             string
		currentAttemptID string
		incomingAttempt  string
	}{
		{name: "stale mismatch", currentAttemptID: "attempt-new", incomingAttempt: "attempt-old"},
		{name: "legacy empty persisted attempt", currentAttemptID: "", incomingAttempt: "attempt-old"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			submission := matchingSubmission()
			submission.CurrentAttemptID = tt.currentAttemptID
			subRepo := &fakeSubmissionRepo{submission: submission}
			resultRepo := &fakeSubmissionResultRepo{}
			eventHub := &fakeSubmissionEventHub{}
			uc := NewApplyJudgeResultUseCase(subRepo, resultRepo, &fakeTxManager{}, eventHub, nil)

			err := uc.Execute(context.Background(), pkgjudge.ResultMessage{
				SubmissionID: 77,
				AttemptID:    tt.incomingAttempt,
				Status:       string(entity.StatusAccepted),
			})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if len(subRepo.updates) != 0 || resultRepo.calls != 0 {
				t.Fatalf("stale/legacy result must not write, updates=%d replacements=%d", len(subRepo.updates), resultRepo.calls)
			}
			if len(eventHub.events) != 0 {
				t.Fatalf("stale/legacy result must not publish, events=%d", len(eventHub.events))
			}
		})
	}
}

func TestApplyJudgeResultRejectsInvalidTransportFieldsBeforeTransaction(t *testing.T) {
	tx := &fakeTxManager{}
	eventHub := &fakeSubmissionEventHub{}
	uc := NewApplyJudgeResultUseCase(&fakeSubmissionRepo{submission: matchingSubmission()}, &fakeSubmissionResultRepo{}, tx, eventHub, nil)

	if err := uc.Execute(context.Background(), pkgjudge.ResultMessage{SubmissionID: 77, Status: string(entity.StatusAccepted)}); !errors.Is(err, domain.ErrInvalidJudgeResult) {
		t.Fatalf("Execute() error = %v, want invalid judge result", err)
	}
	if tx.called {
		t.Fatal("transaction must not start for missing incoming attempt")
	}
	if len(eventHub.events) != 0 {
		t.Fatalf("invalid transport fields must not publish, events=%d", len(eventHub.events))
	}
}

func TestApplyJudgeResultRejectsNonTerminalOrUnknownStatus(t *testing.T) {
	for _, status := range []string{"PENDING", "JUDGING", "BOGUS"} {
		t.Run(status, func(t *testing.T) {
			tx := &fakeTxManager{}
			eventHub := &fakeSubmissionEventHub{}
			uc := NewApplyJudgeResultUseCase(&fakeSubmissionRepo{submission: matchingSubmission()}, &fakeSubmissionResultRepo{}, tx, eventHub, nil)

			err := uc.Execute(context.Background(), pkgjudge.ResultMessage{SubmissionID: 77, AttemptID: "attempt-77", Status: status})
			if !errors.Is(err, domain.ErrInvalidSubmissionStatus) {
				t.Fatalf("Execute() error = %v, want invalid status", err)
			}
			if tx.called {
				t.Fatal("transaction must not start for invalid status")
			}
			if len(eventHub.events) != 0 {
				t.Fatalf("invalid status must not publish, events=%d", len(eventHub.events))
			}
		})
	}
}

func TestApplyJudgeResultDuplicateMatchingResultIsAckNoopAfterTerminalCommit(t *testing.T) {
	subRepo := &fakeSubmissionRepo{submission: matchingSubmission()}
	resultRepo := &fakeSubmissionResultRepo{}
	eventHub := &fakeSubmissionEventHub{}
	uc := NewApplyJudgeResultUseCase(subRepo, resultRepo, &fakeTxManager{}, eventHub, nil)
	msg := pkgjudge.ResultMessage{SubmissionID: 77, AttemptID: "attempt-77", Status: string(entity.StatusAccepted)}

	if err := uc.Execute(context.Background(), msg); err != nil {
		t.Fatalf("first Execute() error = %v", err)
	}
	if err := uc.Execute(context.Background(), msg); err != nil {
		t.Fatalf("second Execute() error = %v", err)
	}
	if len(subRepo.updates) != 1 || resultRepo.calls != 1 || resultRepo.attemptID != "attempt-77" {
		t.Fatalf("duplicate should no-op after terminal commit, updates=%d calls=%d attempt=%q", len(subRepo.updates), resultRepo.calls, resultRepo.attemptID)
	}
	if len(eventHub.events) != 1 {
		t.Fatalf("duplicate should not publish a second event, events=%d", len(eventHub.events))
	}
}

func TestApplyJudgeResultPropagatesRepositoryErrors(t *testing.T) {
	wantErr := errors.New("replace failed")
	eventHub := &fakeSubmissionEventHub{}
	uc := NewApplyJudgeResultUseCase(
		&fakeSubmissionRepo{submission: matchingSubmission()},
		&fakeSubmissionResultRepo{replaceErr: wantErr},
		&fakeTxManager{},
		eventHub,
		nil,
	)

	err := uc.Execute(context.Background(), pkgjudge.ResultMessage{SubmissionID: 77, AttemptID: "attempt-77", Status: string(entity.StatusAccepted)})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
	if len(eventHub.events) != 0 {
		t.Fatalf("repository error must not publish, events=%d", len(eventHub.events))
	}
}

func TestApplyJudgeResultDoesNotPublishOnTransactionCommitFailure(t *testing.T) {
	wantErr := errors.New("commit failed")
	eventHub := &fakeSubmissionEventHub{}
	uc := NewApplyJudgeResultUseCase(
		&fakeSubmissionRepo{submission: matchingSubmission()},
		&fakeSubmissionResultRepo{},
		&fakeTxManager{commitErr: wantErr},
		eventHub,
		nil,
	)

	err := uc.Execute(context.Background(), pkgjudge.ResultMessage{SubmissionID: 77, AttemptID: "attempt-77", Status: string(entity.StatusAccepted)})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
	if len(eventHub.events) != 0 {
		t.Fatalf("commit failure must not publish, events=%d", len(eventHub.events))
	}
}

func TestApplyJudgeResultAlreadyTerminalSubmissionIsAckNoop(t *testing.T) {
	submission := matchingSubmission()
	submission.Status = entity.StatusAccepted
	subRepo := &fakeSubmissionRepo{submission: submission}
	resultRepo := &fakeSubmissionResultRepo{}
	eventHub := &fakeSubmissionEventHub{}
	uc := NewApplyJudgeResultUseCase(subRepo, resultRepo, &fakeTxManager{}, eventHub, nil)

	err := uc.Execute(context.Background(), pkgjudge.ResultMessage{
		SubmissionID: 77,
		AttemptID:    "attempt-77",
		Status:       string(entity.StatusAccepted),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(subRepo.updates) != 0 || resultRepo.calls != 0 || len(eventHub.events) != 0 {
		t.Fatalf("terminal duplicate must be no-op, updates=%d replacements=%d events=%d", len(subRepo.updates), resultRepo.calls, len(eventHub.events))
	}
}

func TestApplyJudgeResultPreservesContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	eventHub := &fakeSubmissionEventHub{}
	uc := NewApplyJudgeResultUseCase(&fakeSubmissionRepo{submission: matchingSubmission()}, &fakeSubmissionResultRepo{}, &fakeTxManager{}, eventHub, nil)

	err := uc.Execute(ctx, pkgjudge.ResultMessage{SubmissionID: 77, AttemptID: "attempt-77", Status: string(entity.StatusAccepted)})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Execute() error = %v, want context canceled", err)
	}
	if len(eventHub.events) != 0 {
		t.Fatalf("canceled context must not publish, events=%d", len(eventHub.events))
	}
}

func stringPointer(value string) *string {
	return &value
}

func sameStringPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
