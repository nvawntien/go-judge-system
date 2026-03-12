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

// everyone can get this problem
type getProblemUseCase struct {
	problemRepo  outbound.ProblemRepository
	testCaseRepo outbound.TestCaseRepository
	logger       *zap.Logger
}

func NewGetProblemUseCase(problemRepo outbound.ProblemRepository, testCaseRepo outbound.TestCaseRepository, logger *zap.Logger) inbound.GetProblemUseCase {
	return &getProblemUseCase{problemRepo: problemRepo, testCaseRepo: testCaseRepo, logger: logger}
}

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

	testCases, err := uc.testCaseRepo.GetByProblemID(ctx, problem.ID)
	if err != nil {
		uc.logger.Error("failed to get test cases", zap.Error(err))
		return dto.ProblemDetailResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	// Public: only show example test cases
	examples := make([]dto.TestCaseResponse, 0)
	for _, tc := range testCases {
		if tc.IsExample {
			examples = append(examples, usecase.MapTestCaseToResponse(tc))
		}
	}

	return dto.ProblemDetailResponse{
		ProblemResponse: usecase.MapProblemToResponse(problem, false),
		TestCases:       examples,
	}, nil
}

// only admin can get by id
func (uc *getProblemUseCase) ExecuteAdmin(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest) (dto.ProblemDetailResponse, error) {
	problem, err := uc.problemRepo.GetByID(ctx, params.ID)
	if err != nil {
		if !errors.Is(err, domain.ErrProblemNotFound) {
			return dto.ProblemDetailResponse{}, domain.ErrInternalServer.Wrap(err)
		}
		return dto.ProblemDetailResponse{}, domain.ErrProblemNotFound
	}

	if !claims.CanManage(problem.AuthorID) {
		return dto.ProblemDetailResponse{}, domain.ErrNotOwner
	}

	testCases, err := uc.testCaseRepo.GetByProblemID(ctx, problem.ID)
	if err != nil {
		uc.logger.Error("failed to get test cases", zap.Error(err))
		return dto.ProblemDetailResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	all := make([]dto.TestCaseResponse, 0, len(testCases))
	for _, tc := range testCases {
		all = append(all, usecase.MapTestCaseToResponse(tc))
	}

	return dto.ProblemDetailResponse{
		ProblemResponse: usecase.MapProblemToResponse(problem, true),
		TestCases:       all,
	}, nil
}
