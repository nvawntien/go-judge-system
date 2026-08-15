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

type listMyProblemRepository struct {
	outbound.ProblemRepository
	authorID string
}

func (r *listMyProblemRepository) ListByAuthor(_ context.Context, authorID string, _, _ int, _, _, _ string) ([]*entity.Problem, error) {
	r.authorID = authorID
	return []*entity.Problem{{ID: 1, AuthorID: authorID, IsHidden: true}}, nil
}

func (r *listMyProblemRepository) CountByAuthor(_ context.Context, authorID, _, _, _ string) (int64, error) {
	r.authorID = authorID
	return 1, nil
}

func TestListMyProblemsRoleAndOwnershipMatrix(t *testing.T) {
	t.Parallel()
	for _, role := range []rbac.Role{rbac.RoleUser, rbac.RoleContributor, rbac.RoleModerator, rbac.RoleAdmin} {
		role := role
		t.Run(string(role), func(t *testing.T) {
			t.Parallel()
			repository := &listMyProblemRepository{}
			response, err := NewListMyProblemsUseCase(repository).Execute(
				context.Background(), auth.Claims{Role: role, UserID: "actor"}, dto.ListProblemsRequest{},
			)
			if !role.AtLeast(rbac.RoleContributor) {
				if !errors.Is(err, domain.ErrForbidden) || repository.authorID != "" {
					t.Fatalf("error/author = %v/%q, want forbidden and no query", err, repository.authorID)
				}
				return
			}
			if err != nil || repository.authorID != "actor" || len(response.Items) != 1 || response.Items[0].AuthorID != "actor" {
				t.Fatalf("response/error/author = %+v/%v/%q", response, err, repository.authorID)
			}
		})
	}
}
