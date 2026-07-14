package problem

import (
	"context"
	"errors"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/problem/internal/application/dto"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"
)

type deleteProblemUseCase struct {
	problemRepo outbound.ProblemRepository
}

func NewDeleteProblemUseCase(problemRepo outbound.ProblemRepository) inbound.DeleteProblemUseCase {
	return &deleteProblemUseCase{
		problemRepo: problemRepo,
	}
}

func (uc *deleteProblemUseCase) Execute(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest) error {
	if !claims.Role.AtLeast(rbac.RoleContributor) {
		return domain.ErrForbidden
	}

	problem, err := uc.problemRepo.GetByID(ctx, params.ID)
	if err != nil {
		if errors.Is(err, domain.ErrProblemNotFound) {
			return domain.ErrProblemNotFound
		}
		return domain.ErrInternalServer.Wrap(err)
	}

	if !claims.Role.AtLeast(rbac.RoleModerator) {
		if problem.AuthorID != claims.UserID {
			return domain.ErrForbidden
		}
		if !problem.IsHidden {
			return domain.ErrForbidden
		}
	}

	if err := uc.problemRepo.Delete(ctx, params.ID); err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	return nil
}
