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

type publishProblemUseCase struct {
	problemRepo outbound.ProblemRepository
	logger      *zap.Logger
}

func NewPublishProblemUseCase(problemRepo outbound.ProblemRepository, logger *zap.Logger) inbound.PublishProblemUseCase {
	return &publishProblemUseCase{problemRepo: problemRepo, logger: logger}
}

func (uc *publishProblemUseCase) Execute(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest) error {
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

	problem.Publish()

	if err := uc.problemRepo.Update(ctx, problem); err != nil {
		uc.logger.Error("failed to publish problem", zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}
	return nil
}
