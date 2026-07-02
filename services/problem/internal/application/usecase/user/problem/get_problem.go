package problem

import (
	"context"
	"errors"
	"strings"

	"go-judge-system/services/problem/internal/application/dto"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/user"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/application/usecase"
	"go-judge-system/services/problem/internal/domain"
)

type getProblemUseCase struct {
	problemRepo outbound.ProblemRepository
}

func NewGetProblemUseCase(problemRepo outbound.ProblemRepository) inbound.GetProblemUseCase {
	return &getProblemUseCase{problemRepo: problemRepo}
}

func (uc *getProblemUseCase) Execute(ctx context.Context, params dto.ProblemSlugRequest) (dto.ProblemDetailResponse, error) {
	slug := strings.TrimSpace(params.Slug)
	if slug == "" {
		return dto.ProblemDetailResponse{}, domain.ErrInvalidProblemSlug
	}

	problem, err := uc.problemRepo.GetBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, domain.ErrProblemNotFound) {
			return dto.ProblemDetailResponse{}, domain.ErrProblemNotFound
		}
		return dto.ProblemDetailResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	if problem.IsHidden {
		return dto.ProblemDetailResponse{}, domain.ErrProblemNotFound
	}

	return dto.ProblemDetailResponse{
		ProblemResponse: usecase.MapProblemToResponse(problem, false),
	}, nil
}
