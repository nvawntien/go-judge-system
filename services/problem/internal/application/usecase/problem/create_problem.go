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

type createProblemUseCase struct {
	problemRepo outbound.ProblemRepository
	logger      *zap.Logger
}

func NewCreateProblemUseCase(problemRepo outbound.ProblemRepository, logger *zap.Logger) inbound.CreateProblemUseCase {
	return &createProblemUseCase{problemRepo: problemRepo, logger: logger}
}

func (uc *createProblemUseCase) Execute(ctx context.Context, claims auth.Claims, req dto.CreateProblemRequest) (dto.CreateProblemResponse, error) {
	if !claims.IsAdmin() {
		return dto.CreateProblemResponse{}, domain.ErrForbidden
	}

	problem := entity.NewProblem(req.Title, req.Slug, req.Description, entity.Difficulty(req.Difficulty), req.TimeLimit, req.MemoryLimit, claims.UserID)

	if err := uc.problemRepo.Create(ctx, problem); err != nil {
		if errors.Is(err, domain.ErrProblemAlreadyExists) {
			return dto.CreateProblemResponse{}, domain.ErrProblemAlreadyExists
		}
		
		uc.logger.Error("failed to create problem", zap.Error(err))
		return dto.CreateProblemResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	return dto.CreateProblemResponse{ID: problem.ID, Slug: problem.Slug}, nil
}
