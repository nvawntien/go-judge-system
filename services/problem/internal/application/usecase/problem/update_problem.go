package problem

import (
	"context"
	"errors"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/problem/internal/application/dto"
	"go-judge-system/services/problem/internal/application/port/inbound"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"
	"go-judge-system/services/problem/internal/domain/entity"

	"go.uber.org/zap"
)

type updateProblemUseCase struct {
	problemRepo outbound.ProblemRepository
	logger      *zap.Logger
}

func NewUpdateProblemUseCase(problemRepo outbound.ProblemRepository, logger *zap.Logger) inbound.UpdateProblemUseCase {
	return &updateProblemUseCase{problemRepo: problemRepo, logger: logger}
}

func (uc *updateProblemUseCase) Execute(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest, body dto.UpdateProblemRequest) error {
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

	if body.Title != nil {
		problem.Title = *body.Title
	}
	if body.NewSlug != nil {
		problem.Slug = *body.NewSlug
	}
	if body.Description != nil {
		problem.Description = *body.Description
	}
	if body.Difficulty != nil {
		problem.Difficulty = entity.Difficulty(*body.Difficulty)
	}
	if body.TimeLimit != nil {
		problem.TimeLimit = *body.TimeLimit
	}
	if body.MemoryLimit != nil {
		problem.MemoryLimit = *body.MemoryLimit
	}

	if err := uc.problemRepo.Update(ctx, problem); err != nil {
		uc.logger.Error("failed to update problem", zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	return nil
}
