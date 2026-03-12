package testcase

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

type deleteTestCaseUseCase struct {
	problemRepo  outbound.ProblemRepository
	testCaseRepo outbound.TestCaseRepository
	logger       *zap.Logger
}

func NewDeleteTestCaseUseCase(problemRepo outbound.ProblemRepository, testCaseRepo outbound.TestCaseRepository, logger *zap.Logger) inbound.DeleteTestCaseUseCase {
	return &deleteTestCaseUseCase{problemRepo: problemRepo, testCaseRepo: testCaseRepo, logger: logger}
}

func (uc *deleteTestCaseUseCase) Execute(ctx context.Context, claims auth.Claims, params dto.TestCaseIDRequest) error {
	tc, err := uc.testCaseRepo.GetByID(ctx, params.ID)
	if err != nil {
		if !errors.Is(err, domain.ErrTestCaseNotFound) {
			return domain.ErrInternalServer.Wrap(err)
		}
		return domain.ErrTestCaseNotFound
	}

	problem, err := uc.problemRepo.GetByID(ctx, tc.ProblemID)
	if err != nil {
		if !errors.Is(err, domain.ErrProblemNotFound) {
			return domain.ErrInternalServer.Wrap(err)
		}
		return domain.ErrProblemNotFound
	}

	if !claims.CanManage(problem.AuthorID) {
		return domain.ErrNotOwner
	}

	if err := uc.testCaseRepo.Delete(ctx, params.ID); err != nil {
		uc.logger.Error("failed to delete test case", zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	return nil
}
