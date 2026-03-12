package problem

import (
	"context"
	"errors"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/problem/internal/application/dto"
	"go-judge-system/services/problem/internal/application/port/inbound"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"

	"go.uber.org/zap"
)

type deleteProblemUseCase struct {
	problemRepo outbound.ProblemRepository
	logger      *zap.Logger
}

func NewDeleteProblemUseCase(problemRepo outbound.ProblemRepository, logger *zap.Logger) inbound.DeleteProblemUseCase {
	return &deleteProblemUseCase{problemRepo: problemRepo, logger: logger}
}

func (uc *deleteProblemUseCase) Execute(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest) error {
	problem, err := uc.problemRepo.GetByID(ctx, params.ID)
	if err != nil {
		if !errors.Is(err, domain.ErrProblemNotFound) {
			return domain.ErrInternalServer.Wrap(err)
		}
		return domain.ErrProblemNotFound
	}

	if !claims.CanManage(problem.AuthorID) {
		return domain.ErrNotOwner
	}

	if err := uc.problemRepo.Delete(ctx, problem.ID); err != nil {
		uc.logger.Error("failed to delete problem", zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	return nil
}
