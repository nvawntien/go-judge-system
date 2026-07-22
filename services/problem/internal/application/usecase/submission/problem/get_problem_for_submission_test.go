package problem

import (
	"context"
	"errors"
	"testing"

	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"
	"go-judge-system/services/problem/internal/domain/entity"
)

type fakeProblemRepository struct {
	outbound.ProblemRepository
	problem *entity.Problem
	err     error
	calls   int
}

func (r *fakeProblemRepository) GetByID(context.Context, int64) (*entity.Problem, error) {
	r.calls++
	return r.problem, r.err
}

func TestGetProblemForSubmissionPublishedProblem(t *testing.T) {
	repo := &fakeProblemRepository{problem: &entity.Problem{
		ID:        42,
		Title:     "Two Sum",
		TitleSlug: "two-sum",
		IsHidden:  false,
	}}

	result, err := NewGetProblemForSubmissionUseCase(repo).Execute(context.Background(), 42)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.ProblemID != 42 || result.Title != "Two Sum" || result.Slug != "two-sum" {
		t.Fatalf("result = %+v", result)
	}
	if repo.calls != 1 {
		t.Fatalf("repository calls = %d, want 1", repo.calls)
	}
}

func TestGetProblemForSubmissionRejectsInvalidID(t *testing.T) {
	repo := &fakeProblemRepository{}
	_, err := NewGetProblemForSubmissionUseCase(repo).Execute(context.Background(), 0)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("Execute() error = %v, want invalid input", err)
	}
	if repo.calls != 0 {
		t.Fatalf("repository calls = %d, want 0", repo.calls)
	}
}

func TestGetProblemForSubmissionTreatsMissingAndHiddenAsNotFound(t *testing.T) {
	tests := []struct {
		name string
		repo *fakeProblemRepository
	}{
		{name: "missing", repo: &fakeProblemRepository{err: domain.ErrProblemNotFound}},
		{name: "hidden", repo: &fakeProblemRepository{problem: &entity.Problem{ID: 42, IsHidden: true}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGetProblemForSubmissionUseCase(tt.repo).Execute(context.Background(), 42)
			if !errors.Is(err, domain.ErrProblemNotFound) {
				t.Fatalf("Execute() error = %v, want problem not found", err)
			}
		})
	}
}

func TestGetProblemForSubmissionWrapsRepositoryFailure(t *testing.T) {
	wantErr := errors.New("database unavailable")
	repo := &fakeProblemRepository{err: wantErr}

	_, err := NewGetProblemForSubmissionUseCase(repo).Execute(context.Background(), 42)
	if !errors.Is(err, domain.ErrInternalServer) || !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestGetProblemForSubmissionPreservesContextErrors(t *testing.T) {
	for _, wantErr := range []error{context.Canceled, context.DeadlineExceeded} {
		repo := &fakeProblemRepository{err: wantErr}
		_, err := NewGetProblemForSubmissionUseCase(repo).Execute(context.Background(), 42)
		if !errors.Is(err, wantErr) {
			t.Fatalf("Execute() error = %v, want %v", err, wantErr)
		}
	}
}
