package problem

import (
	"context"
	"errors"
	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/problem/internal/application/dto"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/application/usecase"
	"go-judge-system/services/problem/internal/domain"
)

type publishProblemUseCase struct {
	problemRepo outbound.ProblemRepository
	cache       outbound.SubmissionProblemCache
}

func NewPublishProblemUseCase(problemRepo outbound.ProblemRepository) inbound.PublishProblemUseCase {
	return newPublishProblemUseCase(problemRepo, nil)
}

func NewCachedPublishProblemUseCase(problemRepo outbound.ProblemRepository, cache outbound.SubmissionProblemCache) inbound.PublishProblemUseCase {
	return newPublishProblemUseCase(problemRepo, cache)
}

func newPublishProblemUseCase(problemRepo outbound.ProblemRepository, cache outbound.SubmissionProblemCache) inbound.PublishProblemUseCase {
	return &publishProblemUseCase{problemRepo: problemRepo, cache: cache}
}

func (uc *publishProblemUseCase) Execute(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest) (dto.ProblemDetailResponse, error) {
	if !claims.Role.AtLeast(rbac.RoleModerator) {
		return dto.ProblemDetailResponse{}, domain.ErrForbidden
	}

	problem, err := uc.problemRepo.GetByID(ctx, params.ID)
	if err != nil {
		if errors.Is(err, domain.ErrProblemNotFound) {
			return dto.ProblemDetailResponse{}, domain.ErrProblemNotFound
		}
		return dto.ProblemDetailResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	if hasInactiveTags(problem.Tags) {
		return dto.ProblemDetailResponse{}, domain.ErrProblemContainsInactiveTags
	}

	problem.Publish()

	if err := uc.problemRepo.Update(ctx, problem); err != nil {
		return dto.ProblemDetailResponse{}, domain.ErrInternalServer.Wrap(err)
	}
	invalidateSubmissionProblemCache(ctx, uc.cache, problem.ID)

	return dto.ProblemDetailResponse{
		ProblemResponse: usecase.MapProblemToResponse(problem, true),
	}, nil
}
