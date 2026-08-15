package testcase

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
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
}

func (r *accessMatrixProblemRepository) GetByID(context.Context, int64) (*entity.Problem, error) {
	copy := *r.problem
	return &copy, nil
}

type accessMatrixTestCaseRepository struct {
	outbound.TestCaseRepository
	deleted bool
}

func (r *accessMatrixTestCaseRepository) GetByProblemID(context.Context, int64) (*entity.TestCase, error) {
	return &entity.TestCase{ID: 7, ProblemID: 42, ZipObjectKey: "private-key", TestCount: 2, Version: 1}, nil
}

func (r *accessMatrixTestCaseRepository) DeleteByProblemID(context.Context, int64) error {
	r.deleted = true
	return nil
}

type accessMatrixTestCaseStorage struct {
	outbound.TestCaseStorage
	deleted bool
}

func (s *accessMatrixTestCaseStorage) DeleteTestCase(context.Context, string) error {
	s.deleted = true
	return nil
}

func testcaseClaims(role rbac.Role, userID string) auth.Claims {
	return auth.Claims{Role: role, UserID: userID}
}

func testcaseProblem(hidden bool) *entity.Problem {
	return &entity.Problem{ID: 42, AuthorID: "owner", IsHidden: hidden}
}

func TestGetTestCaseRoleAndOwnershipMatrix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		role    rbac.Role
		actor   string
		hidden  bool
		allowed bool
	}{
		{name: "user", role: rbac.RoleUser, actor: "owner", hidden: true},
		{name: "contributor owner hidden", role: rbac.RoleContributor, actor: "owner", hidden: true, allowed: true},
		{name: "contributor owner published metadata", role: rbac.RoleContributor, actor: "owner", allowed: true},
		{name: "contributor non-owner", role: rbac.RoleContributor, actor: "other", hidden: true},
		{name: "moderator", role: rbac.RoleModerator, actor: "moderator", allowed: true},
		{name: "admin", role: rbac.RoleAdmin, actor: "admin", allowed: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			response, err := NewGetTestCaseUseCase(
				&accessMatrixProblemRepository{problem: testcaseProblem(test.hidden)},
				&accessMatrixTestCaseRepository{},
			).Execute(context.Background(), testcaseClaims(test.role, test.actor), dto.ProblemIDRequest{ID: 42})
			if test.allowed {
				serialized, marshalErr := json.Marshal(response)
				if err != nil || marshalErr != nil || response.ProblemID != 42 || strings.Contains(string(serialized), "zip_object_key") {
					t.Fatalf("response/error = %+v/%v", response, err)
				}
				return
			}
			if !errors.Is(err, domain.ErrForbidden) {
				t.Fatalf("error = %v, want forbidden", err)
			}
		})
	}
}

func TestUploadTestCaseRoleOwnershipAndPublicationMatrix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		role    rbac.Role
		actor   string
		hidden  bool
		allowed bool
	}{
		{name: "user", role: rbac.RoleUser, actor: "owner", hidden: true},
		{name: "contributor owner hidden", role: rbac.RoleContributor, actor: "owner", hidden: true, allowed: true},
		{name: "contributor owner published", role: rbac.RoleContributor, actor: "owner"},
		{name: "contributor non-owner", role: rbac.RoleContributor, actor: "other", hidden: true},
		{name: "moderator", role: rbac.RoleModerator, actor: "moderator", allowed: true},
		{name: "admin", role: rbac.RoleAdmin, actor: "admin", allowed: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewUploadTestCaseUseCase(
				&accessMatrixProblemRepository{problem: testcaseProblem(test.hidden)},
				&accessMatrixTestCaseRepository{},
				&accessMatrixTestCaseStorage{},
			).Execute(context.Background(), testcaseClaims(test.role, test.actor), dto.ProblemIDRequest{ID: 42}, dto.UploadTestCaseRequest{})
			if test.allowed {
				if !errors.Is(err, domain.ErrInvalidTestCase) {
					t.Fatalf("error = %v, want invalid testcase after authorization", err)
				}
				return
			}
			if !errors.Is(err, domain.ErrForbidden) {
				t.Fatalf("error = %v, want forbidden", err)
			}
		})
	}
}

func TestDeleteTestCaseRoleOwnershipAndPublicationMatrix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		role    rbac.Role
		actor   string
		hidden  bool
		allowed bool
	}{
		{name: "user", role: rbac.RoleUser, actor: "owner", hidden: true},
		{name: "contributor owner hidden", role: rbac.RoleContributor, actor: "owner", hidden: true, allowed: true},
		{name: "contributor owner published", role: rbac.RoleContributor, actor: "owner"},
		{name: "contributor non-owner", role: rbac.RoleContributor, actor: "other", hidden: true},
		{name: "moderator", role: rbac.RoleModerator, actor: "moderator", allowed: true},
		{name: "admin", role: rbac.RoleAdmin, actor: "admin", allowed: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repository := &accessMatrixTestCaseRepository{}
			storage := &accessMatrixTestCaseStorage{}
			err := NewDeleteTestCaseUseCase(
				&accessMatrixProblemRepository{problem: testcaseProblem(test.hidden)}, repository, storage,
			).Execute(context.Background(), testcaseClaims(test.role, test.actor), dto.ProblemIDRequest{ID: 42})
			if test.allowed {
				if err != nil || !repository.deleted || !storage.deleted {
					t.Fatalf("error/deleted/storage = %v/%t/%t", err, repository.deleted, storage.deleted)
				}
				return
			}
			if !errors.Is(err, domain.ErrForbidden) || repository.deleted || storage.deleted {
				t.Fatalf("error/deleted/storage = %v/%t/%t, want forbidden/no mutation", err, repository.deleted, storage.deleted)
			}
		})
	}
}
