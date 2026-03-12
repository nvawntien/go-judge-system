package testcase

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

type createTestCaseUseCase struct {
	problemRepo  outbound.ProblemRepository
	testCaseRepo outbound.TestCaseRepository
	logger       *zap.Logger
}

func NewCreateTestCaseUseCase(problemRepo outbound.ProblemRepository, testCaseRepo outbound.TestCaseRepository, logger *zap.Logger) inbound.CreateTestCaseUseCase {
	return &createTestCaseUseCase{problemRepo: problemRepo, testCaseRepo: testCaseRepo, logger: logger}
}

func (uc *createTestCaseUseCase) Execute(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest, body dto.CreateTestCaseRequest) (dto.CreateTestCaseResponse, error) {
	problem, err := uc.problemRepo.GetByID(ctx, params.ID)
	if err != nil {
		if !errors.Is(err, domain.ErrProblemNotFound) {
			return dto.CreateTestCaseResponse{}, domain.ErrInternalServer.Wrap(err)
		}
		return dto.CreateTestCaseResponse{}, domain.ErrProblemNotFound
	}

	if !claims.CanManage(problem.AuthorID) {
		return dto.CreateTestCaseResponse{}, domain.ErrNotOwner
	}

	tc := entity.NewTestCase(problem.ID, body.Input, body.ExpectedOutput, body.IsExample, body.Order)

	if err := uc.testCaseRepo.Create(ctx, tc); err != nil {
		uc.logger.Error("failed to create test case", zap.Error(err))
		return dto.CreateTestCaseResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	return dto.CreateTestCaseResponse{ID: tc.ID}, nil
}
