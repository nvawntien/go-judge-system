package problem

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-judge-system/pkg/rbac"
	inbound "go-judge-system/services/problem/internal/application/port/inbound"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"
	"go-judge-system/services/problem/internal/domain/entity"
)

type fakeProblemRepository struct {
	outbound.ProblemRepository
	problem   *entity.Problem
	err       error
	problemID int64
	calls     int
}

func (r *fakeProblemRepository) GetByID(_ context.Context, problemID int64) (*entity.Problem, error) {
	r.problemID = problemID
	r.calls++
	return r.problem, r.err
}

func canonicalProblem(hidden bool, authorID string) *entity.Problem {
	return &entity.Problem{
		ID:        42,
		Title:     "Two Sum",
		TitleSlug: "two-sum",
		AuthorID:  authorID,
		IsHidden:  hidden,
	}
}

func getProblemRequest(role rbac.Role, actorUserID string) inbound.GetProblemRequest {
	return inbound.GetProblemRequest{ProblemID: 42, ActorUserID: actorUserID, ActorRole: role}
}

func TestGetProblemAccessMatrix(t *testing.T) {
	tests := []struct {
		name      string
		role      rbac.Role
		actorID   string
		hidden    bool
		authorID  string
		wantAllow bool
	}{
		{name: "user public", role: rbac.RoleUser, actorID: "user-1", wantAllow: true},
		{name: "user hidden", role: rbac.RoleUser, actorID: "user-1", hidden: true},
		{name: "contributor public", role: rbac.RoleContributor, actorID: "contributor-a", wantAllow: true},
		{name: "contributor own hidden", role: rbac.RoleContributor, actorID: "contributor-a", hidden: true, authorID: "contributor-a", wantAllow: true},
		{name: "contributor other hidden", role: rbac.RoleContributor, actorID: "contributor-a", hidden: true, authorID: "contributor-b"},
		{name: "moderator public", role: rbac.RoleModerator, actorID: "moderator-1", wantAllow: true},
		{name: "moderator hidden", role: rbac.RoleModerator, actorID: "moderator-1", hidden: true, wantAllow: true},
		{name: "admin public", role: rbac.RoleAdmin, actorID: "admin-1", wantAllow: true},
		{name: "admin hidden", role: rbac.RoleAdmin, actorID: "admin-1", hidden: true, wantAllow: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeProblemRepository{problem: canonicalProblem(tt.hidden, tt.authorID)}
			got, err := NewGetProblemUseCase(repo).Execute(context.Background(), getProblemRequest(tt.role, tt.actorID))
			if tt.wantAllow {
				if err != nil {
					t.Fatalf("Execute() error = %v", err)
				}
				if got.ID != 42 || got.Title != "Two Sum" || got.Slug != "two-sum" {
					t.Fatalf("metadata = %+v", got)
				}
			} else if !errors.Is(err, domain.ErrProblemNotFound) {
				t.Fatalf("Execute() error = %v, want problem not found", err)
			}
			if repo.calls != 1 || repo.problemID != 42 {
				t.Fatalf("repository calls/id = %d/%d", repo.calls, repo.problemID)
			}
		})
	}
}

func TestGetProblemMissingAndDeletedAreNotFoundForEveryRole(t *testing.T) {
	roles := []rbac.Role{rbac.RoleUser, rbac.RoleContributor, rbac.RoleModerator, rbac.RoleAdmin}
	deletedAt := time.Now()

	for _, role := range roles {
		t.Run(string(role)+" missing", func(t *testing.T) {
			repo := &fakeProblemRepository{err: domain.ErrProblemNotFound}
			_, err := NewGetProblemUseCase(repo).Execute(context.Background(), getProblemRequest(role, "actor-1"))
			if !errors.Is(err, domain.ErrProblemNotFound) {
				t.Fatalf("Execute() error = %v, want problem not found", err)
			}
		})

		t.Run(string(role)+" deleted", func(t *testing.T) {
			problem := canonicalProblem(false, "actor-1")
			problem.DeletedAt = &deletedAt
			repo := &fakeProblemRepository{problem: problem}
			_, err := NewGetProblemUseCase(repo).Execute(context.Background(), getProblemRequest(role, "actor-1"))
			if !errors.Is(err, domain.ErrProblemNotFound) {
				t.Fatalf("Execute() error = %v, want problem not found", err)
			}
		})
	}
}

func TestGetProblemValidatesActorBeforeRepository(t *testing.T) {
	tests := []struct {
		name string
		req  inbound.GetProblemRequest
		want error
	}{
		{name: "invalid problem ID", req: inbound.GetProblemRequest{ActorUserID: "actor", ActorRole: rbac.RoleUser}, want: domain.ErrInvalidInput},
		{name: "missing actor user ID", req: inbound.GetProblemRequest{ProblemID: 42, ActorRole: rbac.RoleUser}, want: domain.ErrActorUnauthenticated},
		{name: "missing actor role", req: inbound.GetProblemRequest{ProblemID: 42, ActorUserID: "actor"}, want: domain.ErrActorUnauthenticated},
		{name: "unsupported actor role", req: inbound.GetProblemRequest{ProblemID: 42, ActorUserID: "actor", ActorRole: rbac.Role("owner")}, want: domain.ErrPermissionDenied},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeProblemRepository{}
			_, err := NewGetProblemUseCase(repo).Execute(context.Background(), tt.req)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.want)
			}
			if repo.calls != 0 {
				t.Fatalf("repository calls = %d, want 0", repo.calls)
			}
		})
	}
}

func TestGetProblemMapsRepositoryFailures(t *testing.T) {
	wantErr := errors.New("database unavailable")
	repo := &fakeProblemRepository{err: wantErr}
	_, err := NewGetProblemUseCase(repo).Execute(context.Background(), getProblemRequest(rbac.RoleUser, "actor"))
	if !errors.Is(err, domain.ErrInternalServer) || !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestGetProblemPreservesContextErrors(t *testing.T) {
	for _, wantErr := range []error{context.Canceled, context.DeadlineExceeded} {
		repo := &fakeProblemRepository{err: wantErr}
		_, err := NewGetProblemUseCase(repo).Execute(context.Background(), getProblemRequest(rbac.RoleUser, "actor"))
		if !errors.Is(err, wantErr) {
			t.Fatalf("Execute() error = %v, want %v", err, wantErr)
		}
	}
}
