package result

import (
	"context"
	"errors"
	"testing"
	"time"

	pkgjudge "go-judge-system/pkg/judge"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"
)

type fakeTxManager struct {
	called bool
}

func (m *fakeTxManager) ExecuteInTx(ctx context.Context, fn func(context.Context) error) error {
	m.called = true
	return fn(ctx)
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
	return nil
}
func (r *fakeSubmissionRepo) List(context.Context, outbound.ListSubmissionsFilter) (outbound.ListSubmissionsResult, error) {
	return outbound.ListSubmissionsResult{}, nil
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
			uc := NewApplyJudgeResultUseCase(subRepo, resultRepo, &fakeTxManager{})
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
			uc := NewApplyJudgeResultUseCase(subRepo, resultRepo, &fakeTxManager{})

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
		})
	}
}

func TestApplyJudgeResultRejectsInvalidTransportFieldsBeforeTransaction(t *testing.T) {
	tx := &fakeTxManager{}
	uc := NewApplyJudgeResultUseCase(&fakeSubmissionRepo{submission: matchingSubmission()}, &fakeSubmissionResultRepo{}, tx)

	if err := uc.Execute(context.Background(), pkgjudge.ResultMessage{SubmissionID: 77, Status: string(entity.StatusAccepted)}); !errors.Is(err, domain.ErrInvalidJudgeResult) {
		t.Fatalf("Execute() error = %v, want invalid judge result", err)
	}
	if tx.called {
		t.Fatal("transaction must not start for missing incoming attempt")
	}
}

func TestApplyJudgeResultRejectsNonTerminalOrUnknownStatus(t *testing.T) {
	for _, status := range []string{"PENDING", "JUDGING", "BOGUS"} {
		t.Run(status, func(t *testing.T) {
			tx := &fakeTxManager{}
			uc := NewApplyJudgeResultUseCase(&fakeSubmissionRepo{submission: matchingSubmission()}, &fakeSubmissionResultRepo{}, tx)

			err := uc.Execute(context.Background(), pkgjudge.ResultMessage{SubmissionID: 77, AttemptID: "attempt-77", Status: status})
			if !errors.Is(err, domain.ErrInvalidSubmissionStatus) {
				t.Fatalf("Execute() error = %v, want invalid status", err)
			}
			if tx.called {
				t.Fatal("transaction must not start for invalid status")
			}
		})
	}
}

func TestApplyJudgeResultDuplicateMatchingResultIsDeterministicReplace(t *testing.T) {
	subRepo := &fakeSubmissionRepo{submission: matchingSubmission()}
	resultRepo := &fakeSubmissionResultRepo{}
	uc := NewApplyJudgeResultUseCase(subRepo, resultRepo, &fakeTxManager{})
	msg := pkgjudge.ResultMessage{SubmissionID: 77, AttemptID: "attempt-77", Status: string(entity.StatusAccepted)}

	if err := uc.Execute(context.Background(), msg); err != nil {
		t.Fatalf("first Execute() error = %v", err)
	}
	if err := uc.Execute(context.Background(), msg); err != nil {
		t.Fatalf("second Execute() error = %v", err)
	}
	if len(subRepo.updates) != 2 || resultRepo.calls != 2 || resultRepo.attemptID != "attempt-77" {
		t.Fatalf("duplicate should converge through replace, updates=%d calls=%d attempt=%q", len(subRepo.updates), resultRepo.calls, resultRepo.attemptID)
	}
}

func TestApplyJudgeResultPropagatesRepositoryErrors(t *testing.T) {
	wantErr := errors.New("replace failed")
	uc := NewApplyJudgeResultUseCase(
		&fakeSubmissionRepo{submission: matchingSubmission()},
		&fakeSubmissionResultRepo{replaceErr: wantErr},
		&fakeTxManager{},
	)

	err := uc.Execute(context.Background(), pkgjudge.ResultMessage{SubmissionID: 77, AttemptID: "attempt-77", Status: string(entity.StatusAccepted)})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

func TestApplyJudgeResultPreservesContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	uc := NewApplyJudgeResultUseCase(&fakeSubmissionRepo{submission: matchingSubmission()}, &fakeSubmissionResultRepo{}, &fakeTxManager{})

	err := uc.Execute(ctx, pkgjudge.ResultMessage{SubmissionID: 77, AttemptID: "attempt-77", Status: string(entity.StatusAccepted)})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Execute() error = %v, want context canceled", err)
	}
}
