package problem

import (
	"context"
	"errors"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/problem/internal/application/dto"
	"go-judge-system/services/problem/internal/application/port/inbound"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/application/usecase"
	"go-judge-system/services/problem/internal/domain"

	"go.uber.org/zap"
)

// getProblemUseCase — public (by slug) and admin (by id) views.
// Examples now live on the Problem entity; no need to query TestCase for public view.
type getProblemUseCase struct {
	problemRepo outbound.ProblemRepository
	logger      *zap.Logger
}

func NewGetProblemUseCase(problemRepo outbound.ProblemRepository, logger *zap.Logger) inbound.GetProblemUseCase {
	return &getProblemUseCase{problemRepo: problemRepo, logger: logger}
}

// Execute — public view (by slug). Returns problem with examples from entity.
func (uc *getProblemUseCase) Execute(ctx context.Context, params dto.ProblemSlugRequest) (dto.ProblemDetailResponse, error) {
	problem, err := uc.problemRepo.GetBySlug(ctx, params.Slug)
	if err != nil {
		if !errors.Is(err, domain.ErrProblemNotFound) {
			return dto.ProblemDetailResponse{}, domain.ErrInternalServer.Wrap(err)
		}
		return dto.ProblemDetailResponse{}, domain.ErrProblemNotFound
	}

	if problem.IsHidden {
		return dto.ProblemDetailResponse{}, domain.ErrProblemNotFound
	}

	return dto.ProblemDetailResponse{
		ProblemResponse: usecase.MapProblemToResponse(problem, false),
	}, nil
}

// ExecuteAdmin — admin view (by id). Returns problem including private fields.
func (uc *getProblemUseCase) ExecuteAdmin(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest) (dto.ProblemDetailResponse, error) {
	problem, err := uc.problemRepo.GetByID(ctx, params.ID)
	if err != nil {
		if !errors.Is(err, domain.ErrProblemNotFound) {
			return dto.ProblemDetailResponse{}, domain.ErrInternalServer.Wrap(err)
		}
		return dto.ProblemDetailResponse{}, domain.ErrProblemNotFound
	}

	if problem.IsHidden && !claims.CanManage(problem.AuthorID) {
		return dto.ProblemDetailResponse{}, domain.ErrProblemNotFound
	}

	return dto.ProblemDetailResponse{
		ProblemResponse: usecase.MapProblemToResponse(problem, true),
	}, nil
}
