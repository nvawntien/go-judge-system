package user

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go-judge-system/pkg/auth"
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
func (r *fakeSubmissionRepository) Update(context.Context, *entity.Submission) error { return nil }
func (r *fakeSubmissionRepository) ListByUser(context.Context, string, int, int, string, string) ([]*entity.Submission, error) {
	return nil, nil
}
func (r *fakeSubmissionRepository) CountByUser(context.Context, string, string, string) (int64, error) {
	return 0, nil
}
func (r *fakeSubmissionRepository) ListByProblem(context.Context, int64, int, int, string, string) ([]*entity.Submission, error) {
	return nil, nil
}
func (r *fakeSubmissionRepository) CountByProblem(context.Context, int64, string, string) (int64, error) {
	return 0, nil
}
func (r *fakeSubmissionRepository) ListAll(context.Context, int, int, *int64, string, string, string) ([]*entity.Submission, error) {
	return nil, nil
}
func (r *fakeSubmissionRepository) CountAll(context.Context, *int64, string, string, string) (int64, error) {
	return 0, nil
}

type fakeJudgePublisher struct {
	publishErr error
	called     bool
	published  *entity.Submission
	metadata   outbound.JudgeJobMetadata
}

func (p *fakeJudgePublisher) Publish(
	ctx context.Context,
	submission *entity.Submission,
	metadata outbound.JudgeJobMetadata,
) error {
	p.called = true
	p.metadata = metadata
	if p.publishErr != nil {
		return p.publishErr
	}
	p.published = submission
	ctx.Value(transactionStateKey{}).(*transactionState).outbox = true
	return nil
}

type fakeProblemReader struct {
	problem outbound.ProblemForSubmission
	err     error
	calls   int
	order   *[]string
}

func (r *fakeProblemReader) GetForSubmission(
	_ context.Context,
	_ int64,
) (outbound.ProblemForSubmission, error) {
	r.calls++
	if r.order != nil {
		*r.order = append(*r.order, "problem")
	}
	return r.problem, r.err
}

func validProblemReader(problemID int64) *fakeProblemReader {
	return &fakeProblemReader{problem: outbound.ProblemForSubmission{
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
	uc := NewCreateSubmissionUseCase(repo, tx, publisher, validProblemReader(req.ProblemID))
	return uc.Execute(context.Background(), auth.Claims{UserID: "user-1", Username: "alice"}, req)
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
			if got.ID != 77 || got.ProblemID != 42 || got.Language != language || got.Status != "PENDING" || got.CreatedAt.IsZero() {
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
			if publisher.published == nil || publisher.published.ID != 77 {
				t.Fatal("publisher must receive the database-generated submission ID")
			}
			if publisher.metadata.ProblemSlug != "two-sum" {
				t.Fatalf("problem slug = %q, want canonical slug", publisher.metadata.ProblemSlug)
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
			uc := NewCreateSubmissionUseCase(&fakeSubmissionRepository{}, tx, &fakeJudgePublisher{}, problemReader)
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
	uc := NewCreateSubmissionUseCase(repo, tx, publisher, problemReader)

	got, err := uc.Execute(
		context.Background(),
		auth.Claims{UserID: "trusted-user", Username: "trusted-name"},
		dto.CreateSubmissionRequest{ProblemID: 42, Language: "GO", SourceCode: "package main"},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if problemReader.calls != 1 {
		t.Fatalf("ProblemReader calls = %d, want 1", problemReader.calls)
	}
	if len(order) != 2 || order[0] != "problem" || order[1] != "transaction" {
		t.Fatalf("call order = %v, want [problem transaction]", order)
	}
	if got.ProblemID != 84 || repo.created.ProblemID != 84 {
		t.Fatalf("canonical problem ID response/stored = %d/%d, want 84", got.ProblemID, repo.created.ProblemID)
	}
	if repo.created.ProblemName != "Two Sum" {
		t.Fatalf("ProblemName = %q, want canonical title", repo.created.ProblemName)
	}
	if repo.created.UserID != "trusted-user" || repo.created.Username != "trusted-name" {
		t.Fatalf("ownership = %q/%q", repo.created.UserID, repo.created.Username)
	}
	if repo.created.Status != entity.StatusPending {
		t.Fatalf("status = %q, want %q", repo.created.Status, entity.StatusPending)
	}
}

func TestCreateSubmissionProblemValidationFailureStartsNoTransaction(t *testing.T) {
	for _, wantErr := range []error{domain.ErrProblemNotFound, domain.ErrProblemServiceUnavailable} {
		t.Run(wantErr.Error(), func(t *testing.T) {
			problemReader := &fakeProblemReader{err: wantErr}
			tx := &fakeTransactionManager{}
			repo := &fakeSubmissionRepository{}
			publisher := &fakeJudgePublisher{}
			uc := NewCreateSubmissionUseCase(repo, tx, publisher, problemReader)

			_, err := uc.Execute(
				context.Background(),
				auth.Claims{UserID: "user-1"},
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
