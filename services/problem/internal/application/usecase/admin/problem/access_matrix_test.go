package problem

import (
	"context"
	"errors"
	"testing"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/problem/internal/application/dto"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"
	"go-judge-system/services/problem/internal/domain/entity"
)

type accessMatrixProblemRepository struct {
	outbound.ProblemRepository
	problem *entity.Problem
	created *entity.Problem
	updated *entity.Problem
	deleted int64
}

func (r *accessMatrixProblemRepository) Create(_ context.Context, problem *entity.Problem) error {
	r.created = problem
	problem.ID = 101
	return nil
}

func (r *accessMatrixProblemRepository) GetByID(_ context.Context, _ int64) (*entity.Problem, error) {
	if r.problem == nil {
		return nil, domain.ErrProblemNotFound
	}
	copy := *r.problem
	return &copy, nil
}

func (r *accessMatrixProblemRepository) GetBySlug(_ context.Context, _ string) (*entity.Problem, error) {
	return nil, domain.ErrProblemNotFound
}

func (r *accessMatrixProblemRepository) Update(_ context.Context, problem *entity.Problem) error {
	r.updated = problem
	return nil
}

func (r *accessMatrixProblemRepository) Delete(_ context.Context, id int64) error {
	r.deleted = id
	return nil
}

func (r *accessMatrixProblemRepository) List(_ context.Context, _, _ int, _, _, _ string, _ bool) ([]*entity.Problem, error) {
	return []*entity.Problem{r.problem}, nil
}

func (r *accessMatrixProblemRepository) Count(_ context.Context, _, _, _ string, _ bool) (int64, error) {
	return 1, nil
}

type accessMatrixTestCaseRepository struct {
	outbound.TestCaseRepository
}

func (r *accessMatrixTestCaseRepository) GetByProblemID(context.Context, int64) (*entity.TestCase, error) {
	return nil, domain.ErrTestCaseNotFound
}

func matrixClaims(role rbac.Role, userID string) auth.Claims {
	return auth.Claims{Role: role, UserID: userID}
}

func matrixProblem(authorID string, hidden bool) *entity.Problem {
	return &entity.Problem{
		ID:           42,
		Title:        "Owned problem",
		TitleSlug:    "owned-problem",
		Description:  "Description",
		InputFormat:  "Input",
		OutputFormat: "Output",
		Difficulty:   entity.Easy,
		AuthorID:     authorID,
		IsHidden:     hidden,
	}
}

func TestCreateProblemRoleMatrix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		role    rbac.Role
		allowed bool
	}{
		{role: rbac.RoleUser},
		{role: rbac.RoleContributor, allowed: true},
		{role: rbac.RoleModerator, allowed: true},
		{role: rbac.RoleAdmin, allowed: true},
	}

	for _, test := range tests {
		t.Run(string(test.role), func(t *testing.T) {
			t.Parallel()
			repository := &accessMatrixProblemRepository{}
			useCase := NewCreateProblemUseCase(repository, nil)
			_, err := useCase.Execute(context.Background(), matrixClaims(test.role, "actor"), dto.CreateProblemRequest{
				Title: "New problem", Description: "Description", InputFormat: "Input", OutputFormat: "Output", Difficulty: "easy",
			})
			if !test.allowed {
				if !errors.Is(err, domain.ErrForbidden) || repository.created != nil {
					t.Fatalf("error/created = %v/%v, want forbidden and no create", err, repository.created)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if repository.created == nil || repository.created.AuthorID != "actor" || !repository.created.IsHidden {
				t.Fatalf("created problem = %+v", repository.created)
			}
		})
	}
}

func TestGetProblemRoleAndOwnershipMatrix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		role    rbac.Role
		actor   string
		hidden  bool
		allowed bool
	}{
		{name: "user", role: rbac.RoleUser, actor: "owner", hidden: true},
		{name: "contributor owner draft", role: rbac.RoleContributor, actor: "owner", hidden: true, allowed: true},
		{name: "contributor owner published", role: rbac.RoleContributor, actor: "owner", allowed: true},
		{name: "contributor non-owner", role: rbac.RoleContributor, actor: "other", hidden: true},
		{name: "moderator", role: rbac.RoleModerator, actor: "moderator", allowed: true},
		{name: "admin", role: rbac.RoleAdmin, actor: "admin", allowed: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			useCase := NewGetProblemUseCase(
				&accessMatrixProblemRepository{problem: matrixProblem("owner", test.hidden)},
				&accessMatrixTestCaseRepository{},
			)
			_, err := useCase.Execute(context.Background(), matrixClaims(test.role, test.actor), dto.ProblemIDRequest{ID: 42})
			if test.allowed && err != nil {
				t.Fatal(err)
			}
			if !test.allowed && !errors.Is(err, domain.ErrForbidden) {
				t.Fatalf("error = %v, want forbidden", err)
			}
		})
	}
}

func TestUpdateProblemRoleOwnershipAndPublicationMatrix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		role    rbac.Role
		actor   string
		hidden  bool
		allowed bool
	}{
		{name: "user", role: rbac.RoleUser, actor: "owner", hidden: true},
		{name: "contributor owner draft", role: rbac.RoleContributor, actor: "owner", hidden: true, allowed: true},
		{name: "contributor non-owner", role: rbac.RoleContributor, actor: "other", hidden: true},
		{name: "contributor owner published", role: rbac.RoleContributor, actor: "owner"},
		{name: "moderator other published", role: rbac.RoleModerator, actor: "moderator", allowed: true},
		{name: "admin other published", role: rbac.RoleAdmin, actor: "admin", allowed: true},
	}
	updatedDescription := "Updated description"

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repository := &accessMatrixProblemRepository{problem: matrixProblem("owner", test.hidden)}
			useCase := NewUpdateProblemUseCase(repository, nil)
			_, err := useCase.Execute(
				context.Background(),
				matrixClaims(test.role, test.actor),
				dto.ProblemIDRequest{ID: 42},
				dto.UpdateProblemRequest{Description: &updatedDescription},
			)
			if test.allowed {
				if err != nil || repository.updated == nil || repository.updated.Description != updatedDescription {
					t.Fatalf("error/updated = %v/%+v", err, repository.updated)
				}
				return
			}
			if !errors.Is(err, domain.ErrForbidden) || repository.updated != nil {
				t.Fatalf("error/updated = %v/%+v, want forbidden and no update", err, repository.updated)
			}
		})
	}
}

func TestDeleteProblemRoleOwnershipAndPublicationMatrix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		role    rbac.Role
		actor   string
		hidden  bool
		allowed bool
	}{
		{name: "user", role: rbac.RoleUser, actor: "owner", hidden: true},
		{name: "contributor owner draft", role: rbac.RoleContributor, actor: "owner", hidden: true, allowed: true},
		{name: "contributor non-owner", role: rbac.RoleContributor, actor: "other", hidden: true},
		{name: "contributor owner published", role: rbac.RoleContributor, actor: "owner"},
		{name: "moderator other published", role: rbac.RoleModerator, actor: "moderator", allowed: true},
		{name: "admin other published", role: rbac.RoleAdmin, actor: "admin", allowed: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repository := &accessMatrixProblemRepository{problem: matrixProblem("owner", test.hidden)}
			err := NewDeleteProblemUseCase(repository).Execute(
				context.Background(), matrixClaims(test.role, test.actor), dto.ProblemIDRequest{ID: 42},
			)
			if test.allowed {
				if err != nil || repository.deleted != 42 {
					t.Fatalf("error/deleted = %v/%d", err, repository.deleted)
				}
				return
			}
			if !errors.Is(err, domain.ErrForbidden) || repository.deleted != 0 {
				t.Fatalf("error/deleted = %v/%d, want forbidden and no delete", err, repository.deleted)
			}
		})
	}
}

func TestAdminListRequiresModerator(t *testing.T) {
	t.Parallel()
	for _, role := range []rbac.Role{rbac.RoleUser, rbac.RoleContributor, rbac.RoleModerator, rbac.RoleAdmin} {
		role := role
		t.Run(string(role), func(t *testing.T) {
			t.Parallel()
			_, err := NewListProblemsUseCase(&accessMatrixProblemRepository{problem: matrixProblem("owner", true)}).
				Execute(context.Background(), matrixClaims(role, "actor"), dto.ListProblemsRequest{})
			if role.AtLeast(rbac.RoleModerator) && err != nil {
				t.Fatal(err)
			}
			if !role.AtLeast(rbac.RoleModerator) && !errors.Is(err, domain.ErrForbidden) {
				t.Fatalf("error = %v, want forbidden", err)
			}
		})
	}
}

func TestPublishAndHideRequireModeratorInUseCase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		role    rbac.Role
		allowed bool
	}{
		{role: rbac.RoleUser},
		{role: rbac.RoleContributor},
		{role: rbac.RoleModerator, allowed: true},
		{role: rbac.RoleAdmin, allowed: true},
	}

	for _, test := range tests {
		test := test
		t.Run(string(test.role), func(t *testing.T) {
			t.Parallel()
			claims := matrixClaims(test.role, "actor")

			publishRepo := &accessMatrixProblemRepository{problem: matrixProblem("owner", true)}
			_, publishErr := NewPublishProblemUseCase(publishRepo).Execute(
				context.Background(), claims, dto.ProblemIDRequest{ID: 42},
			)

			hideRepo := &accessMatrixProblemRepository{problem: matrixProblem("owner", false)}
			_, hideErr := NewHiddenProblemUseCase(hideRepo).Execute(
				context.Background(), claims, dto.ProblemIDRequest{ID: 42},
			)

			if test.allowed {
				if publishErr != nil || publishRepo.updated == nil || publishRepo.updated.IsHidden ||
					hideErr != nil || hideRepo.updated == nil || !hideRepo.updated.IsHidden {
					t.Fatalf("publish error/problem=%v/%+v; hide error/problem=%v/%+v", publishErr, publishRepo.updated, hideErr, hideRepo.updated)
				}
				return
			}

			if !errors.Is(publishErr, domain.ErrForbidden) || publishRepo.updated != nil ||
				!errors.Is(hideErr, domain.ErrForbidden) || hideRepo.updated != nil {
				t.Fatalf("publish error/update=%v/%+v; hide error/update=%v/%+v", publishErr, publishRepo.updated, hideErr, hideRepo.updated)
			}
		})
	}
}
