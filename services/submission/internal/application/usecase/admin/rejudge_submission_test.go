package admin

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
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
	submission     *entity.Submission
	updated        *entity.Submission
	updateErr      error
	forUpdateCalls int
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
	r.forUpdateCalls++
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
	created     *entity.SubmissionAttempt
	createErr   error
	createCalls int
}

func (r *fakeRejudgeAttemptRepo) Create(_ context.Context, attempt *entity.SubmissionAttempt) error {
	r.createCalls++
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
	job   pkgjudge.JobMessage
	err   error
	calls int
}

func (p *fakeRejudgePublisher) Publish(_ context.Context, job pkgjudge.JobMessage) error {
	p.calls++
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
	if subRepo.forUpdateCalls != 1 {
		t.Fatalf("locked submission reads = %d, want 1", subRepo.forUpdateCalls)
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

func TestRejudgeAdminSubmissionAllowsModeratorAndAdmin(t *testing.T) {
	for _, claims := range []auth.Claims{
		{UserID: "moderator-1", Role: rbac.RoleModerator},
		{UserID: "admin-1", Role: rbac.RoleAdmin},
	} {
		t.Run(string(claims.Role), func(t *testing.T) {
			uc := NewRejudgeAdminSubmissionUseCase(
				&fakeRejudgeSubmissionRepo{submission: rejudgeSubmissionFixture(entity.StatusAccepted)},
				&fakeRejudgeAttemptRepo{},
				fakeRejudgeTx{},
				&fakeRejudgePublisher{},
				fakeRejudgeAttemptIDs{next: "attempt-rejudge"},
				fakeRejudgeProblemReader{},
			)
			if _, err := uc.Execute(context.Background(), claims, dto.RejudgeAdminSubmissionRequest{SubmissionID: 77}); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
		})
	}
}

func TestRejudgeAdminSubmissionRejectsForbiddenRole(t *testing.T) {
	for _, role := range []rbac.Role{rbac.RoleUser, rbac.RoleContributor} {
		t.Run(string(role), func(t *testing.T) {
			uc := NewRejudgeAdminSubmissionUseCase(
				&fakeRejudgeSubmissionRepo{submission: rejudgeSubmissionFixture(entity.StatusAccepted)},
				&fakeRejudgeAttemptRepo{},
				fakeRejudgeTx{},
				&fakeRejudgePublisher{},
				fakeRejudgeAttemptIDs{next: "attempt-rejudge"},
				fakeRejudgeProblemReader{},
			)
			_, err := uc.Execute(context.Background(), auth.Claims{UserID: "user-1", Role: role}, dto.RejudgeAdminSubmissionRequest{SubmissionID: 77})
			if !errors.Is(err, domain.ErrSubmissionForbidden) {
				t.Fatalf("Execute() error = %v, want forbidden", err)
			}
		})
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

func TestRejudgeAdminSubmissionRejectsActiveSubmissionWithoutSideEffects(t *testing.T) {
	for _, status := range []entity.Status{entity.StatusPending, entity.StatusJudging} {
		for _, claims := range []auth.Claims{
			{UserID: "moderator-1", Role: rbac.RoleModerator},
			{UserID: "admin-1", Role: rbac.RoleAdmin},
		} {
			t.Run(string(status)+"/"+string(claims.Role), func(t *testing.T) {
				attempts := &fakeRejudgeAttemptRepo{}
				publisher := &fakeRejudgePublisher{}
				uc := NewRejudgeAdminSubmissionUseCase(
					&fakeRejudgeSubmissionRepo{submission: rejudgeSubmissionFixture(status)},
					attempts,
					fakeRejudgeTx{},
					publisher,
					fakeRejudgeAttemptIDs{next: "attempt-rejudge"},
					fakeRejudgeProblemReader{},
				)
				_, err := uc.Execute(context.Background(), claims, dto.RejudgeAdminSubmissionRequest{SubmissionID: 77})
				if !errors.Is(err, domain.ErrSubmissionRejudgeConflict) {
					t.Fatalf("Execute() error = %v, want conflict", err)
				}
				if attempts.createCalls != 0 || publisher.calls != 0 {
					t.Fatalf("active rejudge created attempt/outbox calls = %d/%d, want 0/0", attempts.createCalls, publisher.calls)
				}
			})
		}
	}
}

func TestRejudgeAdminSubmissionAllowsRejudgeAfterTerminalCompletion(t *testing.T) {
	submission := rejudgeSubmissionFixture(entity.StatusPending)
	repo := &fakeRejudgeSubmissionRepo{submission: submission}
	attempts := &fakeRejudgeAttemptRepo{}
	publisher := &fakeRejudgePublisher{}
	uc := NewRejudgeAdminSubmissionUseCase(
		repo,
		attempts,
		fakeRejudgeTx{},
		publisher,
		fakeRejudgeAttemptIDs{next: "attempt-rejudge"},
		fakeRejudgeProblemReader{},
	)
	claims := auth.Claims{UserID: "moderator-1", Role: rbac.RoleModerator}
	req := dto.RejudgeAdminSubmissionRequest{SubmissionID: submission.ID}

	if _, err := uc.Execute(context.Background(), claims, req); !errors.Is(err, domain.ErrSubmissionRejudgeConflict) {
		t.Fatalf("active rejudge error = %v, want conflict", err)
	}
	repo.submission.Status = entity.StatusAccepted
	if _, err := uc.Execute(context.Background(), claims, req); err != nil {
		t.Fatalf("terminal rejudge error = %v", err)
	}
	if attempts.createCalls != 1 || publisher.calls != 1 {
		t.Fatalf("attempt/outbox calls = %d/%d, want 1/1", attempts.createCalls, publisher.calls)
	}
}

func TestRejudgeAdminSubmissionDifferentSubmissionsRemainIndependent(t *testing.T) {
	claims := auth.Claims{UserID: "moderator-1", Role: rbac.RoleModerator}
	for _, submissionID := range []int64{100, 101, 102} {
		t.Run(fmt.Sprintf("submission-%d", submissionID), func(t *testing.T) {
			submission := rejudgeSubmissionFixture(entity.StatusAccepted)
			submission.ID = submissionID
			attempts := &fakeRejudgeAttemptRepo{}
			publisher := &fakeRejudgePublisher{}
			uc := NewRejudgeAdminSubmissionUseCase(
				&fakeRejudgeSubmissionRepo{submission: submission},
				attempts,
				fakeRejudgeTx{},
				publisher,
				fakeRejudgeAttemptIDs{next: fmt.Sprintf("attempt-%d", submissionID)},
				fakeRejudgeProblemReader{},
			)
			if _, err := uc.Execute(context.Background(), claims, dto.RejudgeAdminSubmissionRequest{SubmissionID: submissionID}); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if attempts.createCalls != 1 || publisher.calls != 1 {
				t.Fatalf("attempt/outbox calls = %d/%d, want 1/1", attempts.createCalls, publisher.calls)
			}
		})
	}
}

type serializedRejudgeTx struct {
	mu sync.Mutex
}

func (tx *serializedRejudgeTx) ExecuteInTx(ctx context.Context, fn func(context.Context) error) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	return fn(ctx)
}

type concurrentRejudgeSubmissionRepo struct {
	mu         sync.Mutex
	submission *entity.Submission
	updates    int
}

func (r *concurrentRejudgeSubmissionRepo) Create(context.Context, *entity.Submission) error {
	return nil
}

func (r *concurrentRejudgeSubmissionRepo) GetByID(context.Context, int64) (*entity.Submission, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.submission == nil {
		return nil, domain.ErrSubmissionNotFound
	}
	copy := *r.submission
	return &copy, nil
}

func (r *concurrentRejudgeSubmissionRepo) GetByIDForUpdate(ctx context.Context, id int64) (*entity.Submission, error) {
	return r.GetByID(ctx, id)
}

func (r *concurrentRejudgeSubmissionRepo) Update(_ context.Context, submission *entity.Submission) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *submission
	r.submission = &copy
	r.updates++
	return nil
}

func (r *concurrentRejudgeSubmissionRepo) List(context.Context, outbound.ListSubmissionsFilter) (outbound.ListSubmissionsResult, error) {
	return outbound.ListSubmissionsResult{}, nil
}

func (r *concurrentRejudgeSubmissionRepo) ResultSummaries(context.Context, []int64) (map[int64]outbound.SubmissionResultSummary, error) {
	return map[int64]outbound.SubmissionResultSummary{}, nil
}

type concurrentRejudgeAttemptRepo struct {
	mu       sync.Mutex
	attempts []*entity.SubmissionAttempt
}

func (r *concurrentRejudgeAttemptRepo) Create(_ context.Context, attempt *entity.SubmissionAttempt) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *attempt
	r.attempts = append(r.attempts, &copy)
	return nil
}

func (r *concurrentRejudgeAttemptRepo) GetByAttemptID(context.Context, string) (*entity.SubmissionAttempt, error) {
	return nil, domain.ErrSubmissionNotFound
}

func (r *concurrentRejudgeAttemptRepo) MarkCompleted(context.Context, string, entity.Status, *int, *int, *string) error {
	return nil
}

type concurrentRejudgePublisher struct {
	mu   sync.Mutex
	jobs []pkgjudge.JobMessage
}

func (p *concurrentRejudgePublisher) Publish(_ context.Context, job pkgjudge.JobMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.jobs = append(p.jobs, job)
	return nil
}

type concurrentRejudgeAttemptIDs struct {
	next atomic.Int64
}

func (g *concurrentRejudgeAttemptIDs) NewAttemptID() string {
	return fmt.Sprintf("attempt-concurrent-%d", g.next.Add(1))
}

func TestRejudgeAdminSubmissionConcurrentRequestsCreateOneAttemptAndOutboxJob(t *testing.T) {
	const requests = 20
	submission := rejudgeSubmissionFixture(entity.StatusAccepted)
	repo := &concurrentRejudgeSubmissionRepo{submission: submission}
	attempts := &concurrentRejudgeAttemptRepo{}
	publisher := &concurrentRejudgePublisher{}
	uc := NewRejudgeAdminSubmissionUseCase(
		repo,
		attempts,
		&serializedRejudgeTx{},
		publisher,
		&concurrentRejudgeAttemptIDs{},
		fakeRejudgeProblemReader{},
	)

	start := make(chan struct{})
	errs := make(chan error, requests)
	var wg sync.WaitGroup
	for range requests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := uc.Execute(
				context.Background(),
				auth.Claims{UserID: "moderator-1", Role: rbac.RoleModerator},
				dto.RejudgeAdminSubmissionRequest{SubmissionID: submission.ID},
			)
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	successes := 0
	conflicts := 0
	for err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, domain.ErrSubmissionRejudgeConflict):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent rejudge error: %v", err)
		}
	}
	if successes != 1 || conflicts != requests-1 {
		t.Fatalf("successes/conflicts = %d/%d, want 1/%d", successes, conflicts, requests-1)
	}

	repo.mu.Lock()
	current := *repo.submission
	updates := repo.updates
	repo.mu.Unlock()
	attempts.mu.Lock()
	createdAttempts := append([]*entity.SubmissionAttempt(nil), attempts.attempts...)
	attempts.mu.Unlock()
	publisher.mu.Lock()
	jobs := append([]pkgjudge.JobMessage(nil), publisher.jobs...)
	publisher.mu.Unlock()
	if updates != 1 || len(createdAttempts) != 1 || len(jobs) != 1 {
		t.Fatalf("submission updates/attempts/outbox jobs = %d/%d/%d, want 1/1/1", updates, len(createdAttempts), len(jobs))
	}
	if current.Status != entity.StatusPending || current.CurrentAttemptID != createdAttempts[0].AttemptID || jobs[0].AttemptID != current.CurrentAttemptID {
		t.Fatalf("current submission/attempt/job mismatch: %+v / %+v / %+v", current, createdAttempts[0], jobs[0])
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
