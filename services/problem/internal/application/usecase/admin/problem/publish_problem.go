package problem

import (
	"context"
	"errors"
	"go-judge-system/services/problem/internal/application/dto"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/application/usecase"
	"go-judge-system/services/problem/internal/domain"
)

type publishProblemUseCase struct {
	problemRepo outbound.ProblemRepository
}

func NewPublishProblemUseCase(problemRepo outbound.ProblemRepository) inbound.PublishProblemUseCase {
	return &publishProblemUseCase{problemRepo: problemRepo}
}

func (uc *publishProblemUseCase) Execute(ctx context.Context, params dto.ProblemIDRequest) (dto.ProblemDetailResponse, error) {
	problem, err := uc.problemRepo.GetByID(ctx, params.ID)
	if err != nil {
		if errors.Is(err, domain.ErrProblemNotFound) {
			return dto.ProblemDetailResponse{}, domain.ErrProblemNotFound
		}
		return dto.ProblemDetailResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	problem.Publish()

	if err := uc.problemRepo.Update(ctx, problem); err != nil {
		return dto.ProblemDetailResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	return dto.ProblemDetailResponse{
		ProblemResponse: usecase.MapProblemToResponse(problem, true),
	}, nil
}
