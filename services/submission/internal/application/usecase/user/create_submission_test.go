package user

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go-judge-system/pkg/auth"
	pkgjudge "go-judge-system/pkg/judge"
	"go-judge-system/pkg/rbac"
	"go-judge-system/pkg/response"
	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"
)

type transactionStateKey struct{}

type transactionState struct {
	submission *entity.Submission
	outbox     bool
}

type fakeTransactionManager struct {
	called              bool
	committedSubmission *entity.Submission
	committedOutbox     bool
	rolledBack          bool
	order               *[]string
}

func (m *fakeTransactionManager) ExecuteInTx(ctx context.Context, fn func(context.Context) error) error {
	m.called = true
	if m.order != nil {
		*m.order = append(*m.order, "transaction")
	}
	state := &transactionState{}
	err := fn(context.WithValue(ctx, transactionStateKey{}, state))
	if err != nil {
		m.rolledBack = true
		return err
	}
	m.committedSubmission = state.submission
	m.committedOutbox = state.outbox
	return nil
}

type fakeSubmissionRepository struct {
	createErr error
	created   *entity.Submission
}

func (r *fakeSubmissionRepository) Create(ctx context.Context, submission *entity.Submission) error {
	if r.createErr != nil {
		return r.createErr
	}
	submission.ID = 77
	r.created = submission
	state := ctx.Value(transactionStateKey{}).(*transactionState)
	copy := *submission
	state.submission = &copy
	return nil
}

func (r *fakeSubmissionRepository) GetByID(context.Context, int64) (*entity.Submission, error) {
	return nil, nil
}
func (r *fakeSubmissionRepository) GetByIDForUpdate(context.Context, int64) (*entity.Submission, error) {
	return nil, nil
}
func (r *fakeSubmissionRepository) Update(context.Context, *entity.Submission) error { return nil }
func (r *fakeSubmissionRepository) List(
	context.Context,
	outbound.ListSubmissionsFilter,
) (outbound.ListSubmissionsResult, error) {
	return outbound.ListSubmissionsResult{}, nil
}

type fakeJudgePublisher struct {
	publishErr error
	called     bool
	published  pkgjudge.JobMessage
}

func (p *fakeJudgePublisher) Publish(
	ctx context.Context,
	job pkgjudge.JobMessage,
) error {
	p.called = true
	if p.publishErr != nil {
		return p.publishErr
	}
	p.published = job
	ctx.Value(transactionStateKey{}).(*transactionState).outbox = true
	return nil
}

type fakeAttemptIDGenerator struct {
	next string
}

func (g fakeAttemptIDGenerator) NewAttemptID() string {
	if g.next == "" {
		return "attempt-test-1"
	}
	return g.next
}

type fakeProblemReader struct {
	problem   dto.ProblemMetadata
	err       error
	calls     int
	order     *[]string
	problemID int64
	actor     dto.ProblemActor
}

func (r *fakeProblemReader) GetProblem(
	_ context.Context,
	problemID int64,
	actor dto.ProblemActor,
) (dto.ProblemMetadata, error) {
	r.calls++
	r.problemID = problemID
	r.actor = actor
	if r.order != nil {
		*r.order = append(*r.order, "problem")
	}
	return r.problem, r.err
}

func validProblemReader(problemID int64) *fakeProblemReader {
	return &fakeProblemReader{problem: dto.ProblemMetadata{
		ID:    problemID,
		Title: "Two Sum",
		Slug:  "two-sum",
	}}
}

func executeSubmission(
	t *testing.T,
	req dto.CreateSubmissionRequest,
	repo *fakeSubmissionRepository,
	tx *fakeTransactionManager,
	publisher *fakeJudgePublisher,
) (dto.CreateSubmissionResponse, error) {
	t.Helper()
	uc := NewCreateSubmissionUseCase(repo, tx, publisher, fakeAttemptIDGenerator{}, validProblemReader(req.ProblemID))
	return uc.Execute(context.Background(), auth.Claims{UserID: "user-1", Username: "alice", Role: rbac.RoleUser}, req)
}

func TestCreateSubmission_ExecutableLanguages(t *testing.T) {
	for _, language := range []string{"GO", "CPP", "PYTHON", "JAVA"} {
		t.Run(language, func(t *testing.T) {
			repo := &fakeSubmissionRepository{}
			tx := &fakeTransactionManager{}
			publisher := &fakeJudgePublisher{}

			got, err := executeSubmission(t, dto.CreateSubmissionRequest{
				ProblemID:  42,
				Language:   language,
				SourceCode: "  source preserved exactly\n",
			}, repo, tx, publisher)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if got.ID != 77 || got.ProblemID != 42 || got.ProblemTitle != "Two Sum" || got.Language != language || got.Status != "PENDING" || got.CreatedAt.IsZero() {
				t.Fatalf("unexpected response: %+v", got)
			}
			if !tx.committedOutbox || tx.committedSubmission == nil {
				t.Fatal("submission and outbox must commit together")
			}
			if repo.created.UserID != "user-1" || repo.created.Username != "alice" {
				t.Fatalf("ownership = %q/%q, want claims values", repo.created.UserID, repo.created.Username)
			}
			if repo.created.SourceCode != "  source preserved exactly\n" {
				t.Fatalf("source was modified: %q", repo.created.SourceCode)
			}
			if repo.created.ProblemName != "Two Sum" {
				t.Fatalf("problem name = %q, want canonical title", repo.created.ProblemName)
			}
			if publisher.published.SubmissionID != 77 {
				t.Fatal("publisher must receive the database-generated submission ID")
			}
			if repo.created.CurrentAttemptID == "" {
				t.Fatal("submission current attempt ID must be persisted")
			}
			if publisher.published.AttemptID != repo.created.CurrentAttemptID {
				t.Fatalf("published attempt ID = %q, want stored %q", publisher.published.AttemptID, repo.created.CurrentAttemptID)
			}
			if publisher.published.ProblemSlug != "two-sum" {
				t.Fatalf("problem slug = %q, want canonical slug", publisher.published.ProblemSlug)
			}
			if publisher.published.EnqueuedAt.IsZero() {
				t.Fatal("published job must include enqueued_at")
			}
		})
	}
}

func TestCreateSubmission_Validation(t *testing.T) {
	tests := []struct {
		name    string
		claims  auth.Claims
		req     dto.CreateSubmissionRequest
		wantErr error
	}{
		{name: "missing authenticated user", req: dto.CreateSubmissionRequest{ProblemID: 1, Language: "GO", SourceCode: "x"}, wantErr: response.NewAppError(response.CodeUnauthorized, "unauthorized", nil)},
		{name: "invalid problem ID", claims: auth.Claims{UserID: "u"}, req: dto.CreateSubmissionRequest{Language: "GO", SourceCode: "x"}, wantErr: domain.ErrInvalidProblemID},
		{name: "empty source", claims: auth.Claims{UserID: "u"}, req: dto.CreateSubmissionRequest{ProblemID: 1, Language: "GO"}, wantErr: domain.ErrInvalidSourceCode},
		{name: "whitespace source", claims: auth.Claims{UserID: "u"}, req: dto.CreateSubmissionRequest{ProblemID: 1, Language: "GO", SourceCode: " \n\t"}, wantErr: domain.ErrInvalidSourceCode},
		{name: "source over limit", claims: auth.Claims{UserID: "u"}, req: dto.CreateSubmissionRequest{ProblemID: 1, Language: "GO", SourceCode: strings.Repeat("x", entity.MaxSourceCodeBytes+1)}, wantErr: domain.ErrSourceCodeTooLarge},
		{name: "C unsupported", claims: auth.Claims{UserID: "u"}, req: dto.CreateSubmissionRequest{ProblemID: 1, Language: "C", SourceCode: "x"}, wantErr: domain.ErrInvalidLanguage},
		{name: "JavaScript unsupported", claims: auth.Claims{UserID: "u"}, req: dto.CreateSubmissionRequest{ProblemID: 1, Language: "JAVASCRIPT", SourceCode: "x"}, wantErr: domain.ErrInvalidLanguage},
		{name: "lowercase alias unsupported", claims: auth.Claims{UserID: "u"}, req: dto.CreateSubmissionRequest{ProblemID: 1, Language: "go", SourceCode: "x"}, wantErr: domain.ErrInvalidLanguage},
		{name: "unknown language", claims: auth.Claims{UserID: "u"}, req: dto.CreateSubmissionRequest{ProblemID: 1, Language: "RUST", SourceCode: "x"}, wantErr: domain.ErrInvalidLanguage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := &fakeTransactionManager{}
			problemReader := validProblemReader(tt.req.ProblemID)
			uc := NewCreateSubmissionUseCase(&fakeSubmissionRepository{}, tx, &fakeJudgePublisher{}, fakeAttemptIDGenerator{}, problemReader)
			_, err := uc.Execute(context.Background(), tt.claims, tt.req)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}
			if tx.called {
				t.Fatal("transaction must not start for invalid input")
			}
			if problemReader.calls != 0 {
				t.Fatalf("ProblemReader calls = %d, want 0", problemReader.calls)
			}
		})
	}
}

func TestCreateSubmissionValidatesProblemBeforeTransaction(t *testing.T) {
	order := []string{}
	problemReader := validProblemReader(84)
	problemReader.order = &order
	tx := &fakeTransactionManager{order: &order}
	repo := &fakeSubmissionRepository{}
	publisher := &fakeJudgePublisher{}
	uc := NewCreateSubmissionUseCase(repo, tx, publisher, fakeAttemptIDGenerator{next: "attempt-before-tx"}, problemReader)

	got, err := uc.Execute(
		context.Background(),
		auth.Claims{UserID: "trusted-user", Username: "trusted-name", Role: rbac.RoleModerator},
		dto.CreateSubmissionRequest{ProblemID: 42, Language: "GO", SourceCode: "package main"},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if problemReader.calls != 1 {
		t.Fatalf("ProblemReader calls = %d, want 1", problemReader.calls)
	}
	if problemReader.problemID != 42 || problemReader.actor.UserID != "trusted-user" || problemReader.actor.Role != rbac.RoleModerator {
		t.Fatalf("ProblemReader problem/actor = %d/%+v", problemReader.problemID, problemReader.actor)
	}
	if len(order) != 2 || order[0] != "problem" || order[1] != "transaction" {
		t.Fatalf("call order = %v, want [problem transaction]", order)
	}
	if got.ProblemID != 84 || repo.created.ProblemID != 84 {
		t.Fatalf("canonical problem ID response/stored = %d/%d, want 84", got.ProblemID, repo.created.ProblemID)
	}
	if got.ProblemTitle != "Two Sum" {
		t.Fatalf("ProblemTitle = %q, want canonical title", got.ProblemTitle)
	}
	if got.ProblemTitle == problemReader.problem.Slug {
		t.Fatalf("ProblemTitle = %q, must not use canonical slug", got.ProblemTitle)
	}
	if repo.created.ProblemName != "Two Sum" {
		t.Fatalf("ProblemName = %q, want canonical title", repo.created.ProblemName)
	}
	if publisher.published.ProblemSlug != "two-sum" {
		t.Fatalf("ProblemSlug = %q, want canonical slug", publisher.published.ProblemSlug)
	}
	if repo.created.UserID != "trusted-user" || repo.created.Username != "trusted-name" {
		t.Fatalf("ownership = %q/%q", repo.created.UserID, repo.created.Username)
	}
	if repo.created.Status != entity.StatusPending {
		t.Fatalf("status = %q, want %q", repo.created.Status, entity.StatusPending)
	}
	if repo.created.CurrentAttemptID != "attempt-before-tx" || publisher.published.AttemptID != "attempt-before-tx" {
		t.Fatalf("attempt correlation stored/published = %q/%q", repo.created.CurrentAttemptID, publisher.published.AttemptID)
	}
}

func TestCreateSubmissionProblemValidationFailureStartsNoTransaction(t *testing.T) {
	for _, wantErr := range []error{domain.ErrProblemNotFound, domain.ErrProblemActorForbidden, domain.ErrProblemServiceUnavailable} {
		t.Run(wantErr.Error(), func(t *testing.T) {
			problemReader := &fakeProblemReader{err: wantErr}
			tx := &fakeTransactionManager{}
			repo := &fakeSubmissionRepository{}
			publisher := &fakeJudgePublisher{}
			uc := NewCreateSubmissionUseCase(repo, tx, publisher, fakeAttemptIDGenerator{}, problemReader)

			_, err := uc.Execute(
				context.Background(),
				auth.Claims{UserID: "user-1", Role: rbac.RoleUser},
				dto.CreateSubmissionRequest{ProblemID: 42, Language: "GO", SourceCode: "x"},
			)
			if !errors.Is(err, wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, wantErr)
			}
			if problemReader.calls != 1 || tx.called || repo.created != nil || publisher.called {
				t.Fatalf("calls: problem=%d transaction=%t repository=%t publisher=%t", problemReader.calls, tx.called, repo.created != nil, publisher.called)
			}
		})
	}
}

func TestCreateSubmissionPropagatesActorRoleForHiddenProblemAccess(t *testing.T) {
	tests := []struct {
		name string
		role rbac.Role
	}{
		{name: "contributor own hidden", role: rbac.RoleContributor},
		{name: "moderator hidden", role: rbac.RoleModerator},
		{name: "admin hidden", role: rbac.RoleAdmin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problemReader := validProblemReader(42)
			uc := NewCreateSubmissionUseCase(
				&fakeSubmissionRepository{},
				&fakeTransactionManager{},
				&fakeJudgePublisher{},
				fakeAttemptIDGenerator{},
				problemReader,
			)
			_, err := uc.Execute(
				context.Background(),
				auth.Claims{UserID: "actor-1", Username: "actor", Role: tt.role},
				dto.CreateSubmissionRequest{ProblemID: 42, Language: "GO", SourceCode: "x"},
			)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if problemReader.actor.UserID != "actor-1" || problemReader.actor.Role != tt.role {
				t.Fatalf("actor = %+v", problemReader.actor)
			}
		})
	}
}

func TestCreateSubmission_ExactlyMaxSourceBytes(t *testing.T) {
	repo := &fakeSubmissionRepository{}
	tx := &fakeTransactionManager{}
	publisher := &fakeJudgePublisher{}
	_, err := executeSubmission(t, dto.CreateSubmissionRequest{
		ProblemID:  1,
		Language:   "GO",
		SourceCode: strings.Repeat("x", entity.MaxSourceCodeBytes),
	}, repo, tx, publisher)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestCreateSubmission_RepositoryFailureRollsBack(t *testing.T) {
	wantErr := errors.New("insert submission")
	repo := &fakeSubmissionRepository{createErr: wantErr}
	tx := &fakeTransactionManager{}
	publisher := &fakeJudgePublisher{}

	_, err := executeSubmission(t, dto.CreateSubmissionRequest{ProblemID: 1, Language: "GO", SourceCode: "x"}, repo, tx, publisher)
	if !errors.Is(err, domain.ErrInternalServer) || !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v", err)
	}
	if !tx.rolledBack || tx.committedSubmission != nil || tx.committedOutbox {
		t.Fatal("repository failure must leave no committed submission or outbox")
	}
	if publisher.called {
		t.Fatal("publisher must not run after submission insert failure")
	}
}

func TestCreateSubmission_PublisherFailureRollsBack(t *testing.T) {
	wantErr := errors.New("insert outbox")
	repo := &fakeSubmissionRepository{}
	tx := &fakeTransactionManager{}
	publisher := &fakeJudgePublisher{publishErr: wantErr}

	_, err := executeSubmission(t, dto.CreateSubmissionRequest{ProblemID: 1, Language: "GO", SourceCode: "x"}, repo, tx, publisher)
	if !errors.Is(err, domain.ErrInternalServer) || !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v", err)
	}
	if !tx.rolledBack || tx.committedSubmission != nil || tx.committedOutbox {
		t.Fatal("outbox failure must roll back the submission and outbox")
	}
}
