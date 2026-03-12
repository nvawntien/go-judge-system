package testcase

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

type listTestCasesUseCase struct {
	problemRepo  outbound.ProblemRepository
	testCaseRepo outbound.TestCaseRepository
	logger       *zap.Logger
}

func NewListTestCasesUseCase(problemRepo outbound.ProblemRepository, testCaseRepo outbound.TestCaseRepository, logger *zap.Logger) inbound.ListTestCasesUseCase {
	return &listTestCasesUseCase{problemRepo: problemRepo, testCaseRepo: testCaseRepo, logger: logger}
}

func (uc *listTestCasesUseCase) Execute(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest) (dto.TestCaseListResponse, error) {
	problem, err := uc.problemRepo.GetByID(ctx, params.ID)
	if err != nil {
		if !errors.Is(err, domain.ErrProblemNotFound) {
			return dto.TestCaseListResponse{}, domain.ErrInternalServer.Wrap(err)
		}
		return dto.TestCaseListResponse{}, domain.ErrProblemNotFound
	}

	if !claims.CanManage(problem.AuthorID) {
		return dto.TestCaseListResponse{}, domain.ErrNotOwner
	}

	testCases, err := uc.testCaseRepo.GetByProblemID(ctx, problem.ID)
	if err != nil {
		uc.logger.Error("failed to list test cases", zap.Error(err))
		return dto.TestCaseListResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	items := make([]dto.TestCaseResponse, 0, len(testCases))
	for _, tc := range testCases {
		items = append(items, usecase.MapTestCaseToResponse(tc))
	}

	return dto.TestCaseListResponse{Items: items, Total: len(items)}, nil
}
