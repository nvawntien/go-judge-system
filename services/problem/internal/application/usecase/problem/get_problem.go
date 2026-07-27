package problem

import (
	"context"
	"errors"
	"strings"

	"go-judge-system/pkg/rbac"
	inbound "go-judge-system/services/problem/internal/application/port/inbound"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"
	"go-judge-system/services/problem/internal/domain/entity"
)

type getProblemUseCase struct {
	problemRepo outbound.ProblemRepository
}

func NewGetProblemUseCase(problemRepo outbound.ProblemRepository) inbound.GetProblemUseCase {
	return &getProblemUseCase{problemRepo: problemRepo}
}

func (uc *getProblemUseCase) Execute(
	ctx context.Context,
	req inbound.GetProblemRequest,
) (inbound.ProblemMetadata, error) {
	if req.ProblemID <= 0 {
		return inbound.ProblemMetadata{}, domain.ErrInvalidInput
	}

	actorUserID := strings.TrimSpace(req.ActorUserID)
	if actorUserID == "" || req.ActorRole == "" {
		return inbound.ProblemMetadata{}, domain.ErrActorUnauthenticated
	}
	if req.ActorRole.Level() == 0 {
		return inbound.ProblemMetadata{}, domain.ErrPermissionDenied
	}

	problem, err := uc.problemRepo.GetByID(ctx, req.ProblemID)
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return inbound.ProblemMetadata{}, err
		case errors.Is(err, domain.ErrProblemNotFound):
			return inbound.ProblemMetadata{}, domain.ErrProblemNotFound
		default:
			return inbound.ProblemMetadata{}, domain.ErrInternalServer.Wrap(err)
		}
	}
	if problem == nil {
		return inbound.ProblemMetadata{}, domain.ErrInternalServer
	}
	if problem.IsDeleted() || !canAccessProblem(problem, actorUserID, req.ActorRole) {
		return inbound.ProblemMetadata{}, domain.ErrProblemNotFound
	}

	return inbound.ProblemMetadata{
		ID:          problem.ID,
		Title:       problem.Title,
		Slug:        problem.TitleSlug,
		TimeLimit:   problem.TimeLimit,
		MemoryLimit: problem.MemoryLimit,
	}, nil
}

func canAccessProblem(problem *entity.Problem, actorUserID string, actorRole rbac.Role) bool {
	if !problem.IsHidden {
		return true
	}

	switch actorRole {
	case rbac.RoleContributor:
		return problem.AuthorID == actorUserID
	case rbac.RoleModerator, rbac.RoleAdmin:
		return true
	default:
		return false
	}
}
