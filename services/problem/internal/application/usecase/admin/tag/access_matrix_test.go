package tag

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

type tagAccessMatrixRepository struct {
	outbound.TagRepository
	tag              *entity.Tag
	created          bool
	updated          bool
	deactivated      bool
	listAllCalled    bool
	getByIDCallCount int
}

func (r *tagAccessMatrixRepository) Create(_ context.Context, tag *entity.Tag) error {
	r.created = true
	tag.ID = 9
	return nil
}

func (r *tagAccessMatrixRepository) GetByID(_ context.Context, _ uint) (*entity.Tag, error) {
	r.getByIDCallCount++
	if r.tag == nil {
		return nil, domain.ErrTagNotFound
	}
	copy := *r.tag
	return &copy, nil
}

func (r *tagAccessMatrixRepository) Update(_ context.Context, _ *entity.Tag) error {
	r.updated = true
	return nil
}

func (r *tagAccessMatrixRepository) Deactivate(_ context.Context, _ uint) error {
	r.deactivated = true
	return nil
}

func (r *tagAccessMatrixRepository) CountPublishedProblemsByTagID(context.Context, uint) (int64, error) {
	return 0, nil
}

func (r *tagAccessMatrixRepository) ListAll(context.Context) ([]*entity.Tag, error) {
	r.listAllCalled = true
	return []*entity.Tag{r.tag}, nil
}

func tagClaims(role rbac.Role) auth.Claims {
	return auth.Claims{UserID: "actor", Role: role}
}

func activeTag() *entity.Tag {
	return &entity.Tag{ID: 5, Name: "Graphs", Slug: "graphs", IsActive: true}
}

func TestAdminTagListRequiresModerator(t *testing.T) {
	t.Parallel()
	for _, role := range []rbac.Role{rbac.RoleUser, rbac.RoleContributor, rbac.RoleModerator, rbac.RoleAdmin} {
		role := role
		t.Run(string(role), func(t *testing.T) {
			t.Parallel()
			repository := &tagAccessMatrixRepository{tag: activeTag()}
			response, err := NewListTagsUseCase(repository).Execute(context.Background(), tagClaims(role))
			if role.AtLeast(rbac.RoleModerator) {
				if err != nil || !repository.listAllCalled || len(response.Items) != 1 {
					t.Fatalf("error/list/items = %v/%v/%d", err, repository.listAllCalled, len(response.Items))
				}
				return
			}
			if !errors.Is(err, domain.ErrForbidden) || repository.listAllCalled {
				t.Fatalf("error/list = %v/%v, want forbidden/no list", err, repository.listAllCalled)
			}
		})
	}
}

func TestTagMutationRoleMatrix(t *testing.T) {
	t.Parallel()
	operations := []struct {
		name string
		run  func(*tagAccessMatrixRepository, auth.Claims) error
		used func(*tagAccessMatrixRepository) bool
	}{
		{
			name: "create",
			run: func(repository *tagAccessMatrixRepository, claims auth.Claims) error {
				_, err := NewCreateTagUseCase(repository).Execute(context.Background(), claims, dto.CreateTagRequest{Name: "Trees"})
				return err
			},
			used: func(repository *tagAccessMatrixRepository) bool { return repository.created },
		},
		{
			name: "update",
			run: func(repository *tagAccessMatrixRepository, claims auth.Claims) error {
				name := "Shortest paths"
				_, err := NewUpdateTagUseCase(repository).Execute(
					context.Background(), claims, dto.TagIDRequest{ID: 5}, dto.UpdateTagRequest{Name: &name},
				)
				return err
			},
			used: func(repository *tagAccessMatrixRepository) bool { return repository.updated },
		},
		{
			name: "delete",
			run: func(repository *tagAccessMatrixRepository, claims auth.Claims) error {
				return NewDeleteTagUseCase(repository).Execute(context.Background(), claims, dto.TagIDRequest{ID: 5})
			},
			used: func(repository *tagAccessMatrixRepository) bool { return repository.deactivated },
		},
	}

	for _, operation := range operations {
		operation := operation
		t.Run(operation.name, func(t *testing.T) {
			for _, role := range []rbac.Role{rbac.RoleUser, rbac.RoleContributor, rbac.RoleModerator, rbac.RoleAdmin} {
				role := role
				t.Run(string(role), func(t *testing.T) {
					t.Parallel()
					repository := &tagAccessMatrixRepository{tag: activeTag()}
					err := operation.run(repository, tagClaims(role))
					if role.AtLeast(rbac.RoleModerator) {
						if err != nil || !operation.used(repository) {
							t.Fatalf("error/operation = %v/%v, want success/executed", err, operation.used(repository))
						}
						return
					}
					if !errors.Is(err, domain.ErrForbidden) || operation.used(repository) || repository.getByIDCallCount != 0 {
						t.Fatalf("error/operation/get = %v/%v/%d, want forbidden/not executed/no read", err, operation.used(repository), repository.getByIDCallCount)
					}
				})
			}
		})
	}
}
