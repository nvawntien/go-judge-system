package user

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

type fakeGetSubmissionRepository struct {
	*fakeSubmissionRepository
	submission   *entity.Submission
	err          error
	summaryErr   error
	summaries    map[int64]outbound.SubmissionResultSummary
	calls        int
	summaryCalls int
	id           int64
}

func newFakeGetSubmissionRepository(submission *entity.Submission) *fakeGetSubmissionRepository {
	return &fakeGetSubmissionRepository{
		fakeSubmissionRepository: &fakeSubmissionRepository{},
		submission:               submission,
	}
}

func (r *fakeGetSubmissionRepository) GetByID(ctx context.Context, id int64) (*entity.Submission, error) {
	r.calls++
	r.id = id
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return r.submission, r.err
}

func (r *fakeGetSubmissionRepository) ResultSummaries(
	ctx context.Context,
	submissionIDs []int64,
) (map[int64]outbound.SubmissionResultSummary, error) {
	r.summaryCalls++
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.summaryErr != nil {
		return nil, r.summaryErr
	}
	if r.summaries == nil {
		return map[int64]outbound.SubmissionResultSummary{}, nil
	}
	return r.summaries, nil
}

func submissionDetailFixture(ownerID string) *entity.Submission {
	executionTime := 12
	memoryUsed := 4300
	compileOutput := "main.go:8:2: undefined: fmtx"
	return &entity.Submission{
		ID:            77,
		ProblemID:     42,
		ProblemName:   "Two Sum",
		UserID:        ownerID,
		Username:      "owner-name",
		Language:      entity.LanguageGo,
		SourceCode:    "package main\n",
		Status:        entity.StatusCompilationError,
		ExecutionTime: &executionTime,
		MemoryUsed:    &memoryUsed,
		CompileOutput: &compileOutput,
		CreatedAt:     time.Date(2026, 7, 23, 14, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 7, 23, 14, 1, 0, 0, time.UTC),
	}
}

func TestGetSubmissionAccessMatrix(t *testing.T) {
	tests := []struct {
		name        string
		role        rbac.Role
		actorUserID string
		wantErr     error
	}{
		{name: "user reads own submission", role: rbac.RoleUser, actorUserID: "owner"},
		{name: "user reads another submission", role: rbac.RoleUser, actorUserID: "another", wantErr: domain.ErrSubmissionNotFound},
		{name: "contributor reads own submission", role: rbac.RoleContributor, actorUserID: "owner"},
		{name: "contributor reads another submission", role: rbac.RoleContributor, actorUserID: "another", wantErr: domain.ErrSubmissionNotFound},
		{name: "moderator reads own submission", role: rbac.RoleModerator, actorUserID: "owner"},
		{name: "moderator reads another submission", role: rbac.RoleModerator, actorUserID: "another"},
		{name: "admin reads own submission", role: rbac.RoleAdmin, actorUserID: "owner"},
		{name: "admin reads another submission", role: rbac.RoleAdmin, actorUserID: "another"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			submission := submissionDetailFixture("owner")
			repo := newFakeGetSubmissionRepository(submission)
			repo.summaries = map[int64]outbound.SubmissionResultSummary{
				submission.ID: {SubmissionID: submission.ID, Passed: 4, Total: 20},
			}
			uc := NewGetSubmissionUseCase(repo)

			got, err := uc.Execute(
				context.Background(),
				auth.Claims{UserID: tt.actorUserID, Role: tt.role},
				dto.GetSubmissionRequest{SubmissionID: submission.ID},
			)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}
			if repo.calls != 1 || repo.id != submission.ID {
				t.Fatalf("repository calls/id = %d/%d, want 1/%d", repo.calls, repo.id, submission.ID)
			}
			if tt.wantErr != nil {
				if repo.summaryCalls != 0 {
					t.Fatalf("summary calls = %d, want 0 when access denied", repo.summaryCalls)
				}
				if got != (dto.GetSubmissionResponse{}) {
					t.Fatalf("response = %+v, want zero value", got)
				}
				return
			}
			if repo.summaryCalls != 1 {
				t.Fatalf("summary calls = %d, want 1", repo.summaryCalls)
			}

			want := dto.GetSubmissionResponse{
				ID:              submission.ID,
				ProblemID:       submission.ProblemID,
				ProblemTitle:    submission.ProblemName,
				UserID:          submission.UserID,
				Username:        submission.Username,
				Language:        string(submission.Language),
				SourceCode:      submission.SourceCode,
				Status:          string(submission.Status),
				ExecutionTimeMS: submission.ExecutionTime,
				MemoryUsedKB:    submission.MemoryUsed,
				PassedTestCases: intPointer(4),
				TotalTestCases:  intPointer(20),
				CompileOutput:   stringValue(submission.CompileOutput),
				ErrorMessage:    stringValue(submission.CompileOutput),
				CreatedAt:       submission.CreatedAt,
				UpdatedAt:       submission.UpdatedAt,
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("response = %+v, want %+v", got, want)
			}
			if got.ProblemTitle != "Two Sum" {
				t.Fatalf("ProblemTitle = %q, want ProblemName snapshot", got.ProblemTitle)
			}
		})
	}
}

func TestGetSubmissionValidation(t *testing.T) {
	tests := []struct {
		name    string
		claims  auth.Claims
		req     dto.GetSubmissionRequest
		wantErr error
	}{
		{
			name:    "zero submission ID",
			claims:  auth.Claims{UserID: "actor", Role: rbac.RoleUser},
			wantErr: domain.ErrInvalidSubmissionID,
		},
		{
			name:    "negative submission ID",
			claims:  auth.Claims{UserID: "actor", Role: rbac.RoleUser},
			req:     dto.GetSubmissionRequest{SubmissionID: -1},
			wantErr: domain.ErrInvalidSubmissionID,
		},
		{
			name:    "missing actor user ID",
			claims:  auth.Claims{Role: rbac.RoleUser},
			req:     dto.GetSubmissionRequest{SubmissionID: 77},
			wantErr: domain.ErrSubmissionUnauthenticated,
		},
		{
			name:    "missing actor role",
			claims:  auth.Claims{UserID: "actor"},
			req:     dto.GetSubmissionRequest{SubmissionID: 77},
			wantErr: domain.ErrSubmissionUnauthenticated,
		},
		{
			name:    "unsupported actor role",
			claims:  auth.Claims{UserID: "actor", Role: rbac.Role("auditor")},
			req:     dto.GetSubmissionRequest{SubmissionID: 77},
			wantErr: domain.ErrSubmissionForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeGetSubmissionRepository(submissionDetailFixture("actor"))
			uc := NewGetSubmissionUseCase(repo)

			_, err := uc.Execute(context.Background(), tt.claims, tt.req)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}
			if repo.calls != 0 {
				t.Fatalf("repository calls = %d, want 0", repo.calls)
			}
		})
	}
}

func TestGetSubmissionRepositoryErrors(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "missing submission", repoErr: domain.ErrSubmissionNotFound, wantErr: domain.ErrSubmissionNotFound},
		{name: "repository failure", repoErr: databaseErr, wantErr: domain.ErrInternalServer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeGetSubmissionRepository(nil)
			repo.err = tt.repoErr
			uc := NewGetSubmissionUseCase(repo)

			_, err := uc.Execute(
				context.Background(),
				auth.Claims{UserID: "actor", Role: rbac.RoleUser},
				dto.GetSubmissionRequest{SubmissionID: 77},
			)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}
			if repo.calls != 1 {
				t.Fatalf("repository calls = %d, want 1", repo.calls)
			}
			if tt.repoErr == databaseErr && !errors.Is(err, databaseErr) {
				t.Fatalf("Execute() error = %v, want wrapped repository error", err)
			}
		})
	}
}

func TestGetSubmissionSummaryErrorIsMappedSafely(t *testing.T) {
	databaseErr := errors.New("summary unavailable")
	repo := newFakeGetSubmissionRepository(submissionDetailFixture("actor"))
	repo.summaryErr = databaseErr

	_, err := NewGetSubmissionUseCase(repo).Execute(
		context.Background(),
		auth.Claims{UserID: "actor", Role: rbac.RoleUser},
		dto.GetSubmissionRequest{SubmissionID: 77},
	)
	if !errors.Is(err, domain.ErrInternalServer) || !errors.Is(err, databaseErr) {
		t.Fatalf("Execute() error = %v, want safely wrapped summary error", err)
	}
}

func TestGetSubmissionPreservesContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repo := newFakeGetSubmissionRepository(submissionDetailFixture("actor"))
	uc := NewGetSubmissionUseCase(repo)
	_, err := uc.Execute(
		ctx,
		auth.Claims{UserID: "actor", Role: rbac.RoleUser},
		dto.GetSubmissionRequest{SubmissionID: 77},
	)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Execute() error = %v, want context canceled", err)
	}
	if errors.Is(err, domain.ErrInternalServer) {
		t.Fatalf("context cancellation must not be remapped to internal error: %v", err)
	}
}
