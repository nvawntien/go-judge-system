package problem

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/problem/internal/application/dto"
	"go-judge-system/services/problem/internal/domain"
	"go-judge-system/services/problem/internal/domain/entity"
)

type fakeProblemRepository struct {
	nextID   int64
	bySlug   map[string]*entity.Problem
	byAuthor map[string][]*entity.Problem
}

func newFakeProblemRepository() *fakeProblemRepository {
	return &fakeProblemRepository{
		nextID:   1,
		bySlug:   make(map[string]*entity.Problem),
		byAuthor: make(map[string][]*entity.Problem),
	}
}

func (r *fakeProblemRepository) Create(_ context.Context, problem *entity.Problem) error {
	if _, exists := r.bySlug[problem.TitleSlug]; exists {
		return domain.ErrProblemAlreadyExists
	}

	problem.ID = r.nextID
	r.nextID++

	cloned := *problem
	r.bySlug[problem.TitleSlug] = &cloned
	r.byAuthor[problem.AuthorID] = append(r.byAuthor[problem.AuthorID], &cloned)
	return nil
}

func (r *fakeProblemRepository) GetByID(_ context.Context, id int64) (*entity.Problem, error) {
	for _, problem := range r.bySlug {
		if problem.ID == id {
			cloned := *problem
			return &cloned, nil
		}
	}
	return nil, domain.ErrProblemNotFound
}

func (r *fakeProblemRepository) GetBySlug(_ context.Context, slug string) (*entity.Problem, error) {
	problem, ok := r.bySlug[slug]
	if !ok {
		return nil, domain.ErrProblemNotFound
	}

	cloned := *problem
	return &cloned, nil
}

func (r *fakeProblemRepository) Update(_ context.Context, problem *entity.Problem) error {
	cloned := *problem
	r.bySlug[problem.TitleSlug] = &cloned
	return nil
}

func (r *fakeProblemRepository) Delete(_ context.Context, id int64) error {
	for slug, problem := range r.bySlug {
		if problem.ID == id {
			delete(r.bySlug, slug)
			return nil
		}
	}
	return nil
}

func (r *fakeProblemRepository) List(_ context.Context, _, _ int, _, _ string, _ bool) ([]*entity.Problem, error) {
	return nil, nil
}

func (r *fakeProblemRepository) Count(_ context.Context, _, _ string, _ bool) (int64, error) {
	return int64(len(r.bySlug)), nil
}

func (r *fakeProblemRepository) ListByAuthor(_ context.Context, authorID string, _, _ int, _, _ string) ([]*entity.Problem, error) {
	items := r.byAuthor[authorID]
	result := make([]*entity.Problem, 0, len(items))
	for _, item := range items {
		cloned := *item
		result = append(result, &cloned)
	}
	return result, nil
}

func (r *fakeProblemRepository) CountByAuthor(_ context.Context, authorID string, _, _ string) (int64, error) {
	return int64(len(r.byAuthor[authorID])), nil
}

func validCreateProblemRequest() dto.CreateProblemRequest {
	return dto.CreateProblemRequest{
		Title:       "Two Sum",
		Description: "Find two numbers whose sum equals target.",
		Difficulty:  "easy",
		Examples: []dto.ProblemExampleDTO{
			{
				Input:  "nums = [2,7,11,15], target = 9",
				Output: "[0,1]",
			},
		},
		Constraints: []string{"2 <= nums.length <= 10^4"},
		Hints:       []string{"Use a hash map"},
	}
}

func TestCreateProblemUseCaseRejectsUserRole(t *testing.T) {
	repo := newFakeProblemRepository()
	uc := NewCreateProblemUseCase(repo)

	_, err := uc.Execute(context.Background(), auth.Claims{
		UserID: "user-1",
		Role:   rbac.RoleUser,
	}, validCreateProblemRequest())
	if err == nil {
		t.Fatalf("expected forbidden error")
	}
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestCreateProblemUseCaseAllowsContributorModeratorAndAdmin(t *testing.T) {
	testCases := []struct {
		name string
		role rbac.Role
	}{
		{name: "contributor", role: rbac.RoleContributor},
		{name: "moderator", role: rbac.RoleModerator},
		{name: "admin", role: rbac.RoleAdmin},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeProblemRepository()
			uc := NewCreateProblemUseCase(repo)

			res, err := uc.Execute(context.Background(), auth.Claims{
				UserID: "author-1",
				Role:   tc.role,
			}, validCreateProblemRequest())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if res.Slug != "two-sum" {
				t.Fatalf("expected slug two-sum, got %s", res.Slug)
			}
			if res.AuthorID != "author-1" {
				t.Fatalf("expected author id author-1, got %s", res.AuthorID)
			}
			if !res.IsHidden {
				t.Fatalf("expected created problem to be hidden/draft")
			}
			if res.TimeLimit != defaultTimeLimit {
				t.Fatalf("expected default time limit %v, got %v", defaultTimeLimit, res.TimeLimit)
			}
			if res.MemoryLimit != defaultMemoryLimit {
				t.Fatalf("expected default memory limit %d, got %d", defaultMemoryLimit, res.MemoryLimit)
			}
			if res.Difficulty != string(entity.Easy) {
				t.Fatalf("expected difficulty %s, got %s", entity.Easy, res.Difficulty)
			}
			if _, err := time.Parse("2006-01-02T15:04:05Z", res.CreatedAt); err != nil {
				t.Fatalf("expected created_at to be RFC3339-like, got %q", res.CreatedAt)
			}
		})
	}
}

func TestCreateProblemUseCaseValidatesInput(t *testing.T) {
	testCases := []struct {
		name string
		mut  func(*dto.CreateProblemRequest)
	}{
		{
			name: "blank title",
			mut: func(req *dto.CreateProblemRequest) {
				req.Title = "   "
			},
		},
		{
			name: "blank description",
			mut: func(req *dto.CreateProblemRequest) {
				req.Description = "   "
			},
		},
		{
			name: "invalid difficulty",
			mut: func(req *dto.CreateProblemRequest) {
				req.Difficulty = "expert"
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeProblemRepository()
			uc := NewCreateProblemUseCase(repo)
			req := validCreateProblemRequest()
			tc.mut(&req)

			_, err := uc.Execute(context.Background(), auth.Claims{
				UserID: "author-1",
				Role:   rbac.RoleContributor,
			}, req)
			if err == nil {
				t.Fatalf("expected validation error")
			}
		})
	}
}

func TestCreateProblemUseCaseGeneratesUniqueSlugSuffix(t *testing.T) {
	repo := newFakeProblemRepository()
	uc := NewCreateProblemUseCase(repo)
	claims := auth.Claims{
		UserID: "author-1",
		Role:   rbac.RoleContributor,
	}

	first, err := uc.Execute(context.Background(), claims, validCreateProblemRequest())
	if err != nil {
		t.Fatalf("unexpected error creating first problem: %v", err)
	}

	second, err := uc.Execute(context.Background(), claims, validCreateProblemRequest())
	if err != nil {
		t.Fatalf("unexpected error creating second problem: %v", err)
	}

	if first.Slug != "two-sum" {
		t.Fatalf("expected first slug two-sum, got %s", first.Slug)
	}
	if second.Slug != "two-sum-2" {
		t.Fatalf("expected second slug two-sum-2, got %s", second.Slug)
	}
}
