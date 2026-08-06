package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-judge-system/pkg/auth"
	pkgjudge "go-judge-system/pkg/judge"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"
)

type fakeRejudgeSubmissionRepo struct {
	submission *entity.Submission
	updated    *entity.Submission
	updateErr  error
}

func (r *fakeRejudgeSubmissionRepo) Create(context.Context, *entity.Submission) error { return nil }
func (r *fakeRejudgeSubmissionRepo) GetByID(context.Context, int64) (*entity.Submission, error) {
	if r.submission == nil {
		return nil, domain.ErrSubmissionNotFound
	}
	copy := *r.submission
	return &copy, nil
}
func (r *fakeRejudgeSubmissionRepo) GetByIDForUpdate(context.Context, int64) (*entity.Submission, error) {
	if r.submission == nil {
		return nil, domain.ErrSubmissionNotFound
	}
	copy := *r.submission
	return &copy, nil
}
func (r *fakeRejudgeSubmissionRepo) Update(_ context.Context, submission *entity.Submission) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	copy := *submission
	r.updated = &copy
	r.submission = &copy
	return nil
}
func (r *fakeRejudgeSubmissionRepo) List(context.Context, outbound.ListSubmissionsFilter) (outbound.ListSubmissionsResult, error) {
	return outbound.ListSubmissionsResult{}, nil
}
func (r *fakeRejudgeSubmissionRepo) ResultSummaries(context.Context, []int64) (map[int64]outbound.SubmissionResultSummary, error) {
	return map[int64]outbound.SubmissionResultSummary{}, nil
}

type fakeRejudgeAttemptRepo struct {
	created   *entity.SubmissionAttempt
	createErr error
}

func (r *fakeRejudgeAttemptRepo) Create(_ context.Context, attempt *entity.SubmissionAttempt) error {
	if r.createErr != nil {
		return r.createErr
	}
	copy := *attempt
	r.created = &copy
	return nil
}
func (r *fakeRejudgeAttemptRepo) GetByAttemptID(context.Context, string) (*entity.SubmissionAttempt, error) {
	return nil, domain.ErrSubmissionNotFound
}
func (r *fakeRejudgeAttemptRepo) MarkCompleted(context.Context, string, entity.Status, *int, *int, *string) error {
	return nil
}

type fakeRejudgeTx struct{}

func (fakeRejudgeTx) ExecuteInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type rollbackRejudgeTx struct {
	subRepo     *fakeRejudgeSubmissionRepo
	attemptRepo *fakeRejudgeAttemptRepo
	publisher   *fakeRejudgePublisher
}

func (tx rollbackRejudgeTx) ExecuteInTx(ctx context.Context, fn func(context.Context) error) error {
	var submissionSnapshot *entity.Submission
	if tx.subRepo != nil && tx.subRepo.submission != nil {
		copy := *tx.subRepo.submission
		submissionSnapshot = &copy
	}
	var updatedSnapshot *entity.Submission
	if tx.subRepo != nil && tx.subRepo.updated != nil {
		copy := *tx.subRepo.updated
		updatedSnapshot = &copy
	}
	var attemptSnapshot *entity.SubmissionAttempt
	if tx.attemptRepo != nil && tx.attemptRepo.created != nil {
		copy := *tx.attemptRepo.created
		attemptSnapshot = &copy
	}
	var jobSnapshot pkgjudge.JobMessage
	if tx.publisher != nil {
		jobSnapshot = tx.publisher.job
	}

	err := fn(ctx)
	if err != nil {
		if tx.subRepo != nil {
			tx.subRepo.submission = submissionSnapshot
			tx.subRepo.updated = updatedSnapshot
		}
		if tx.attemptRepo != nil {
			tx.attemptRepo.created = attemptSnapshot
		}
		if tx.publisher != nil {
			tx.publisher.job = jobSnapshot
		}
	}
	return err
}

type fakeRejudgePublisher struct {
	job pkgjudge.JobMessage
	err error
}

func (p *fakeRejudgePublisher) Publish(_ context.Context, job pkgjudge.JobMessage) error {
	p.job = job
	return p.err
}

type fakeRejudgeAttemptIDs struct {
	next string
}

func (g fakeRejudgeAttemptIDs) NewAttemptID() string {
	return g.next
}

type fakeRejudgeProblemReader struct {
	err error
}

func (r fakeRejudgeProblemReader) GetProblem(context.Context, int64, dto.ProblemActor) (dto.ProblemMetadata, error) {
	return dto.ProblemMetadata{}, nil
}
func (r fakeRejudgeProblemReader) GetTestCaseMetadata(context.Context, int64) (dto.ProblemTestCaseMetadata, error) {
	if r.err != nil {
		return dto.ProblemTestCaseMetadata{}, r.err
	}
	return dto.ProblemTestCaseMetadata{ProblemID: 42, TestCount: 24, Version: 3}, nil
}

func TestRejudgeAdminSubmissionCreatesNewAttemptAndJob(t *testing.T) {
	submission := rejudgeSubmissionFixture(entity.StatusWrongAnswer)
	subRepo := &fakeRejudgeSubmissionRepo{submission: submission}
	attemptRepo := &fakeRejudgeAttemptRepo{}
	publisher := &fakeRejudgePublisher{}
	uc := NewRejudgeAdminSubmissionUseCase(
		subRepo,
		attemptRepo,
		fakeRejudgeTx{},
		publisher,
		fakeRejudgeAttemptIDs{next: "attempt-rejudge"},
		fakeRejudgeProblemReader{},
	)

	got, err := uc.Execute(context.Background(), auth.Claims{UserID: "moderator-1", Role: rbac.RoleModerator}, dto.RejudgeAdminSubmissionRequest{SubmissionID: 77})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.SubmissionID != 77 || got.AttemptID != "attempt-rejudge" || got.Status != string(entity.StatusPending) {
		t.Fatalf("response = %+v", got)
	}
	if subRepo.updated == nil || subRepo.updated.Status != entity.StatusPending || subRepo.updated.CurrentAttemptID != "attempt-rejudge" {
		t.Fatalf("updated submission = %+v", subRepo.updated)
	}
	if subRepo.updated.ExecutionTime != nil || subRepo.updated.MemoryUsed != nil || subRepo.updated.CompileOutput != nil || subRepo.updated.ErrorMessage != nil {
		t.Fatalf("rejudge did not clear previous terminal fields: %+v", subRepo.updated)
	}
	if attemptRepo.created == nil ||
		attemptRepo.created.AttemptID != "attempt-rejudge" ||
		attemptRepo.created.TriggerType != entity.AttemptTriggerAdminRejudge ||
		attemptRepo.created.TriggeredByUserID == nil ||
		*attemptRepo.created.TriggeredByUserID != "moderator-1" {
		t.Fatalf("created attempt = %+v", attemptRepo.created)
	}
	if publisher.job.SubmissionID != submission.ID ||
		publisher.job.ProblemID != submission.ProblemID ||
		publisher.job.UserID != submission.UserID ||
		publisher.job.Language != string(submission.Language) ||
		publisher.job.SourceCode != submission.SourceCode ||
		publisher.job.AttemptID != "attempt-rejudge" {
		t.Fatalf("published job = %+v", publisher.job)
	}
}

func TestRejudgeAdminSubmissionAttemptInsertFailureRollsBack(t *testing.T) {
	wantErr := errors.New("attempt insert failed")
	submission := rejudgeSubmissionFixture(entity.StatusWrongAnswer)
	subRepo := &fakeRejudgeSubmissionRepo{submission: submission}
	attemptRepo := &fakeRejudgeAttemptRepo{createErr: wantErr}
	publisher := &fakeRejudgePublisher{}
	uc := NewRejudgeAdminSubmissionUseCase(
		subRepo,
		attemptRepo,
		rollbackRejudgeTx{subRepo: subRepo, attemptRepo: attemptRepo, publisher: publisher},
		publisher,
		fakeRejudgeAttemptIDs{next: "attempt-rejudge"},
		fakeRejudgeProblemReader{},
	)

	_, err := uc.Execute(context.Background(), auth.Claims{UserID: "moderator-1", Role: rbac.RoleModerator}, dto.RejudgeAdminSubmissionRequest{SubmissionID: 77})
	if !errors.Is(err, domain.ErrInternalServer) {
		t.Fatalf("Execute() error = %v, want internal wrapper", err)
	}
	assertRejudgeRollback(t, subRepo, attemptRepo, publisher, submission)
}

func TestRejudgeAdminSubmissionUpdateFailureRollsBack(t *testing.T) {
	wantErr := errors.New("submission update failed")
	submission := rejudgeSubmissionFixture(entity.StatusWrongAnswer)
	subRepo := &fakeRejudgeSubmissionRepo{submission: submission, updateErr: wantErr}
	attemptRepo := &fakeRejudgeAttemptRepo{}
	publisher := &fakeRejudgePublisher{}
	uc := NewRejudgeAdminSubmissionUseCase(
		subRepo,
		attemptRepo,
		rollbackRejudgeTx{subRepo: subRepo, attemptRepo: attemptRepo, publisher: publisher},
		publisher,
		fakeRejudgeAttemptIDs{next: "attempt-rejudge"},
		fakeRejudgeProblemReader{},
	)

	_, err := uc.Execute(context.Background(), auth.Claims{UserID: "moderator-1", Role: rbac.RoleModerator}, dto.RejudgeAdminSubmissionRequest{SubmissionID: 77})
	if !errors.Is(err, domain.ErrInternalServer) {
		t.Fatalf("Execute() error = %v, want internal wrapper", err)
	}
	assertRejudgeRollback(t, subRepo, attemptRepo, publisher, submission)
}

func TestRejudgeAdminSubmissionOutboxFailureRollsBack(t *testing.T) {
	wantErr := errors.New("outbox insert failed")
	submission := rejudgeSubmissionFixture(entity.StatusWrongAnswer)
	subRepo := &fakeRejudgeSubmissionRepo{submission: submission}
	attemptRepo := &fakeRejudgeAttemptRepo{}
	publisher := &fakeRejudgePublisher{err: wantErr}
	uc := NewRejudgeAdminSubmissionUseCase(
		subRepo,
		attemptRepo,
		rollbackRejudgeTx{subRepo: subRepo, attemptRepo: attemptRepo, publisher: publisher},
		publisher,
		fakeRejudgeAttemptIDs{next: "attempt-rejudge"},
		fakeRejudgeProblemReader{},
	)

	_, err := uc.Execute(context.Background(), auth.Claims{UserID: "moderator-1", Role: rbac.RoleModerator}, dto.RejudgeAdminSubmissionRequest{SubmissionID: 77})
	if !errors.Is(err, domain.ErrInternalServer) {
		t.Fatalf("Execute() error = %v, want internal wrapper", err)
	}
	assertRejudgeRollback(t, subRepo, attemptRepo, publisher, submission)
}

func TestRejudgeAdminSubmissionRejectsForbiddenRole(t *testing.T) {
	uc := NewRejudgeAdminSubmissionUseCase(
		&fakeRejudgeSubmissionRepo{submission: rejudgeSubmissionFixture(entity.StatusAccepted)},
		&fakeRejudgeAttemptRepo{},
		fakeRejudgeTx{},
		&fakeRejudgePublisher{},
		fakeRejudgeAttemptIDs{next: "attempt-rejudge"},
		fakeRejudgeProblemReader{},
	)
	_, err := uc.Execute(context.Background(), auth.Claims{UserID: "user-1", Role: rbac.RoleContributor}, dto.RejudgeAdminSubmissionRequest{SubmissionID: 77})
	if !errors.Is(err, domain.ErrSubmissionForbidden) {
		t.Fatalf("Execute() error = %v, want forbidden", err)
	}
}

func assertRejudgeRollback(
	t *testing.T,
	subRepo *fakeRejudgeSubmissionRepo,
	attemptRepo *fakeRejudgeAttemptRepo,
	publisher *fakeRejudgePublisher,
	before *entity.Submission,
) {
	t.Helper()
	if subRepo.submission == nil ||
		subRepo.submission.CurrentAttemptID != before.CurrentAttemptID ||
		subRepo.submission.Status != before.Status {
		t.Fatalf("submission after rollback = %+v, want attempt/status %q/%s", subRepo.submission, before.CurrentAttemptID, before.Status)
	}
	if attemptRepo.created != nil {
		t.Fatalf("attempt persisted after rollback: %+v", attemptRepo.created)
	}
	if publisher.job.SubmissionID != 0 {
		t.Fatalf("outbox job persisted after rollback: %+v", publisher.job)
	}
}

func TestRejudgeAdminSubmissionRejectsActiveSubmission(t *testing.T) {
	uc := NewRejudgeAdminSubmissionUseCase(
		&fakeRejudgeSubmissionRepo{submission: rejudgeSubmissionFixture(entity.StatusPending)},
		&fakeRejudgeAttemptRepo{},
		fakeRejudgeTx{},
		&fakeRejudgePublisher{},
		fakeRejudgeAttemptIDs{next: "attempt-rejudge"},
		fakeRejudgeProblemReader{},
	)
	_, err := uc.Execute(context.Background(), auth.Claims{UserID: "mod-1", Role: rbac.RoleModerator}, dto.RejudgeAdminSubmissionRequest{SubmissionID: 77})
	if !errors.Is(err, domain.ErrSubmissionRejudgeConflict) {
		t.Fatalf("Execute() error = %v, want conflict", err)
	}
}

func TestRejudgeAdminSubmissionRejectsMissingTestcase(t *testing.T) {
	uc := NewRejudgeAdminSubmissionUseCase(
		&fakeRejudgeSubmissionRepo{submission: rejudgeSubmissionFixture(entity.StatusAccepted)},
		&fakeRejudgeAttemptRepo{},
		fakeRejudgeTx{},
		&fakeRejudgePublisher{},
		fakeRejudgeAttemptIDs{next: "attempt-rejudge"},
		fakeRejudgeProblemReader{err: domain.ErrSubmissionTestCaseRequired},
	)
	_, err := uc.Execute(context.Background(), auth.Claims{UserID: "mod-1", Role: rbac.RoleModerator}, dto.RejudgeAdminSubmissionRequest{SubmissionID: 77})
	if !errors.Is(err, domain.ErrSubmissionTestCaseRequired) {
		t.Fatalf("Execute() error = %v, want testcase required", err)
	}
}

func rejudgeSubmissionFixture(status entity.Status) *entity.Submission {
	runtime := 18
	memory := 4096
	compileOutput := "old compile"
	errMessage := "old verdict"
	return &entity.Submission{
		ID:               77,
		ProblemID:        42,
		ProblemName:      "Two Sum",
		UserID:           "user-123",
		Username:         "alice",
		Language:         entity.LanguageCPP,
		SourceCode:       "int main(){return 0;}",
		CurrentAttemptID: "attempt-old",
		Status:           status,
		ExecutionTime:    &runtime,
		MemoryUsed:       &memory,
		CompileOutput:    &compileOutput,
		ErrorMessage:     &errMessage,
		CreatedAt:        time.Date(2026, 8, 2, 8, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2026, 8, 2, 8, 1, 0, 0, time.UTC),
	}
}
