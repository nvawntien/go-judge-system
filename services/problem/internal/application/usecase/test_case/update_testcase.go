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

type updateTestCaseUseCase struct {
	problemRepo  outbound.ProblemRepository
	testCaseRepo outbound.TestCaseRepository
	logger       *zap.Logger
}

func NewUpdateTestCaseUseCase(problemRepo outbound.ProblemRepository, testCaseRepo outbound.TestCaseRepository, logger *zap.Logger) inbound.UpdateTestCaseUseCase {
	return &updateTestCaseUseCase{problemRepo: problemRepo, testCaseRepo: testCaseRepo, logger: logger}
}

func (uc *updateTestCaseUseCase) Execute(ctx context.Context, claims auth.Claims, params dto.TestCaseIDRequest, body dto.UpdateTestCaseRequest) error {
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

	if body.Input != nil {
		tc.Input = *body.Input
	}
	if body.ExpectedOutput != nil {
		tc.ExpectedOutput = *body.ExpectedOutput
	}
	if body.IsExample != nil {
		tc.IsExample = *body.IsExample
	}
	if body.Order != nil {
		tc.Order = *body.Order
	}

	if err := uc.testCaseRepo.Update(ctx, tc); err != nil {
		uc.logger.Error("failed to update test case", zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	return nil
}
