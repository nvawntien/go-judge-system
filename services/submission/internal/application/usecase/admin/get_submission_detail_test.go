package admin

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"
)

type fakeGetAdminSubmissionRepository struct {
	submission *entity.Submission
	err        error
	calls      int
	id         int64
}

func (r *fakeGetAdminSubmissionRepository) Create(context.Context, *entity.Submission) error {
	return nil
}

func (r *fakeGetAdminSubmissionRepository) GetByID(ctx context.Context, id int64) (*entity.Submission, error) {
	r.calls++
	r.id = id
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return r.submission, r.err
}

func (r *fakeGetAdminSubmissionRepository) GetByIDForUpdate(context.Context, int64) (*entity.Submission, error) {
	return nil, nil
}

func (r *fakeGetAdminSubmissionRepository) Update(context.Context, *entity.Submission) error {
	return nil
}

func (r *fakeGetAdminSubmissionRepository) List(
	context.Context,
	outbound.ListSubmissionsFilter,
) (outbound.ListSubmissionsResult, error) {
	return outbound.ListSubmissionsResult{}, nil
}

func (r *fakeGetAdminSubmissionRepository) ResultSummaries(
	context.Context,
	[]int64,
) (map[int64]outbound.SubmissionResultSummary, error) {
	return map[int64]outbound.SubmissionResultSummary{}, nil
}

type fakeGetAdminSubmissionResultRepository struct {
	results      []*entity.SubmissionResult
	err          error
	calls        int
	submissionID int64
	attemptID    string
}

func (r *fakeGetAdminSubmissionResultRepository) GetBySubmissionID(
	context.Context,
	int64,
) ([]*entity.SubmissionResult, error) {
	return nil, nil
}

func (r *fakeGetAdminSubmissionResultRepository) GetBySubmissionIDAndAttemptID(
	ctx context.Context,
	submissionID int64,
	attemptID string,
) ([]*entity.SubmissionResult, error) {
	r.calls++
	r.submissionID = submissionID
	r.attemptID = attemptID
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return r.results, r.err
}

func (r *fakeGetAdminSubmissionResultRepository) DeleteBySubmissionID(context.Context, int64) error {
	return nil
}

func (r *fakeGetAdminSubmissionResultRepository) ReplaceBySubmissionIDAndAttemptID(
	context.Context,
	int64,
	string,
	[]*entity.SubmissionResult,
) error {
	return nil
}

func adminSubmissionDetailFixture() *entity.Submission {
	executionTime := 18
	memoryUsed := 4096
	return &entity.Submission{
		ID:               77,
		ProblemID:        42,
		ProblemName:      "Two Sum",
		UserID:           "user-123",
		Username:         "alice",
		Language:         entity.LanguageCPP,
		SourceCode:       "#include <iostream>\nint main() { return 0; }\n",
		CurrentAttemptID: "attempt-current",
		Status:           entity.StatusAccepted,
		ExecutionTime:    &executionTime,
		MemoryUsed:       &memoryUsed,
		CreatedAt:        time.Date(2026, 8, 2, 8, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2026, 8, 2, 8, 1, 0, 0, time.UTC),
	}
}

func TestGetAdminSubmissionDetailAllowedRoles(t *testing.T) {
	for _, role := range []rbac.Role{rbac.RoleModerator, rbac.RoleAdmin} {
		t.Run(string(role), func(t *testing.T) {
			submission := adminSubmissionDetailFixture()
			runtime := 2
			memory := 128
			subRepo := &fakeGetAdminSubmissionRepository{submission: submission}
			resultRepo := &fakeGetAdminSubmissionResultRepository{
				results: []*entity.SubmissionResult{
					{SubmissionID: submission.ID, AttemptID: submission.CurrentAttemptID, TestIndex: 2, Status: entity.ResultAccepted, ExecutionTime: &runtime, MemoryUsed: &memory},
					{SubmissionID: submission.ID, AttemptID: submission.CurrentAttemptID, TestIndex: 1, Status: entity.ResultAccepted, ExecutionTime: &runtime, MemoryUsed: &memory},
				},
			}

			got, err := NewGetAdminSubmissionDetailUseCase(subRepo, resultRepo).Execute(
				context.Background(),
				auth.Claims{UserID: "actor", Role: role},
				dto.GetAdminSubmissionDetailRequest{SubmissionID: submission.ID},
			)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if subRepo.calls != 1 || subRepo.id != submission.ID {
				t.Fatalf("submission repo calls/id = %d/%d", subRepo.calls, subRepo.id)
			}
			if resultRepo.calls != 1 ||
				resultRepo.submissionID != submission.ID ||
				resultRepo.attemptID != submission.CurrentAttemptID {
				t.Fatalf("result repo call = %d/%d/%q", resultRepo.calls, resultRepo.submissionID, resultRepo.attemptID)
			}
			if got.SourceCode != submission.SourceCode ||
				got.CurrentAttemptID != submission.CurrentAttemptID ||
				got.PassedTestCount != 2 ||
				got.ExecutedTestCount != 2 ||
				got.TotalTestCount == nil ||
				*got.TotalTestCount != 2 {
				t.Fatalf("response summary = %+v", got)
			}
			if got.TestResults[0].Index != 1 || got.TestResults[1].Index != 2 {
				t.Fatalf("test results not sorted: %+v", got.TestResults)
			}
		})
	}
}

func TestGetAdminSubmissionDetailRejectsUnauthorizedAndForbiddenRoles(t *testing.T) {
	tests := []struct {
		name    string
		claims  auth.Claims
		wantErr error
	}{
		{name: "missing user ID", claims: auth.Claims{Role: rbac.RoleModerator}, wantErr: domain.ErrSubmissionUnauthenticated},
		{name: "missing role", claims: auth.Claims{UserID: "actor"}, wantErr: domain.ErrSubmissionUnauthenticated},
		{name: "user forbidden", claims: auth.Claims{UserID: "actor", Role: rbac.RoleUser}, wantErr: domain.ErrSubmissionForbidden},
		{name: "contributor forbidden", claims: auth.Claims{UserID: "actor", Role: rbac.RoleContributor}, wantErr: domain.ErrSubmissionForbidden},
		{name: "unknown role forbidden", claims: auth.Claims{UserID: "actor", Role: rbac.Role("auditor")}, wantErr: domain.ErrSubmissionForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subRepo := &fakeGetAdminSubmissionRepository{submission: adminSubmissionDetailFixture()}
			resultRepo := &fakeGetAdminSubmissionResultRepository{}
			_, err := NewGetAdminSubmissionDetailUseCase(subRepo, resultRepo).Execute(
				context.Background(),
				tt.claims,
				dto.GetAdminSubmissionDetailRequest{SubmissionID: 77},
			)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}
			if subRepo.calls != 0 || resultRepo.calls != 0 {
				t.Fatalf("repositories called before auth: submission=%d result=%d", subRepo.calls, resultRepo.calls)
			}
		})
	}
}

func TestGetAdminSubmissionDetailValidationAndRepositoryErrors(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	tests := []struct {
		name    string
		req     dto.GetAdminSubmissionDetailRequest
		repoErr error
		wantErr error
	}{
		{name: "zero submission ID", wantErr: domain.ErrInvalidSubmissionID},
		{name: "negative submission ID", req: dto.GetAdminSubmissionDetailRequest{SubmissionID: -1}, wantErr: domain.ErrInvalidSubmissionID},
		{name: "not found", req: dto.GetAdminSubmissionDetailRequest{SubmissionID: 77}, repoErr: domain.ErrSubmissionNotFound, wantErr: domain.ErrSubmissionNotFound},
		{name: "repository failure", req: dto.GetAdminSubmissionDetailRequest{SubmissionID: 77}, repoErr: databaseErr, wantErr: domain.ErrInternalServer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subRepo := &fakeGetAdminSubmissionRepository{err: tt.repoErr}
			resultRepo := &fakeGetAdminSubmissionResultRepository{}
			_, err := NewGetAdminSubmissionDetailUseCase(subRepo, resultRepo).Execute(
				context.Background(),
				auth.Claims{UserID: "moderator", Role: rbac.RoleModerator},
				tt.req,
			)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}
			if tt.repoErr == databaseErr && !errors.Is(err, databaseErr) {
				t.Fatalf("Execute() error = %v, want wrapped repository error", err)
			}
		})
	}
}

func TestGetAdminSubmissionDetailResultRepositoryFailureIsMappedSafely(t *testing.T) {
	databaseErr := errors.New("result rows unavailable")
	subRepo := &fakeGetAdminSubmissionRepository{submission: adminSubmissionDetailFixture()}
	resultRepo := &fakeGetAdminSubmissionResultRepository{err: databaseErr}

	_, err := NewGetAdminSubmissionDetailUseCase(subRepo, resultRepo).Execute(
		context.Background(),
		auth.Claims{UserID: "admin", Role: rbac.RoleAdmin},
		dto.GetAdminSubmissionDetailRequest{SubmissionID: 77},
	)
	if !errors.Is(err, domain.ErrInternalServer) || !errors.Is(err, databaseErr) {
		t.Fatalf("Execute() error = %v, want safely wrapped result error", err)
	}
}

func TestGetAdminSubmissionDetailCompilationErrorWithoutResultRows(t *testing.T) {
	submission := adminSubmissionDetailFixture()
	compileMessage := "main.cpp:1:1: error: expected declaration"
	submission.Status = entity.StatusCompilationError
	submission.CompileOutput = &compileMessage
	submission.ExecutionTime = nil
	submission.MemoryUsed = nil
	subRepo := &fakeGetAdminSubmissionRepository{submission: submission}
	resultRepo := &fakeGetAdminSubmissionResultRepository{results: []*entity.SubmissionResult{}}

	got, err := NewGetAdminSubmissionDetailUseCase(subRepo, resultRepo).Execute(
		context.Background(),
		auth.Claims{UserID: "moderator", Role: rbac.RoleModerator},
		dto.GetAdminSubmissionDetailRequest{SubmissionID: 77},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	want := dto.GetAdminSubmissionDetailResponse{
		ID:                submission.ID,
		ProblemID:         submission.ProblemID,
		ProblemTitle:      submission.ProblemName,
		UserID:            submission.UserID,
		Username:          submission.Username,
		Language:          string(submission.Language),
		SourceCode:        submission.SourceCode,
		Status:            string(submission.Status),
		CurrentAttemptID:  submission.CurrentAttemptID,
		PassedTestCount:   0,
		ExecutedTestCount: 0,
		TotalTestCount:    intPointer(0),
		CompileMessage:    &compileMessage,
		CreatedAt:         submission.CreatedAt,
		UpdatedAt:         submission.UpdatedAt,
		TestResults:       []dto.AdminSubmissionTestResult{},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("response = %+v, want %+v", got, want)
	}
}

func TestGetAdminSubmissionDetailEarlyTerminationDoesNotInventTotal(t *testing.T) {
	submission := adminSubmissionDetailFixture()
	submission.Status = entity.StatusWrongAnswer
	subRepo := &fakeGetAdminSubmissionRepository{submission: submission}
	resultRepo := &fakeGetAdminSubmissionResultRepository{
		results: []*entity.SubmissionResult{
			{SubmissionID: submission.ID, AttemptID: submission.CurrentAttemptID, TestIndex: 1, Status: entity.ResultAccepted},
			{SubmissionID: submission.ID, AttemptID: submission.CurrentAttemptID, TestIndex: 2, Status: entity.ResultWrongAnswer},
		},
	}

	got, err := NewGetAdminSubmissionDetailUseCase(subRepo, resultRepo).Execute(
		context.Background(),
		auth.Claims{UserID: "admin", Role: rbac.RoleAdmin},
		dto.GetAdminSubmissionDetailRequest{SubmissionID: 77},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.PassedTestCount != 1 || got.ExecutedTestCount != 2 || got.TotalTestCount != nil {
		t.Fatalf("summary = passed %d executed %d total %v, want 1/2/unknown", got.PassedTestCount, got.ExecutedTestCount, got.TotalTestCount)
	}
}
