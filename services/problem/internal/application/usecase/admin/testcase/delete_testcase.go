package testcase

import (
	"context"
	"errors"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/problem/internal/application/dto"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"
)

type deleteTestCaseUseCase struct {
	problemRepo     outbound.ProblemRepository
	testcaseRepo    outbound.TestCaseRepository
	testcaseStorage outbound.TestCaseStorage
}

func NewDeleteTestCaseUseCase(problemRepo outbound.ProblemRepository, testcaseRepo outbound.TestCaseRepository, testcaseStorage outbound.TestCaseStorage) inbound.DeleteTestCaseUseCase {
	return &deleteTestCaseUseCase{
		problemRepo:     problemRepo,
		testcaseRepo:    testcaseRepo,
		testcaseStorage: testcaseStorage,
	}
}

func (uc *deleteTestCaseUseCase) Execute(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest) error {
	if !claims.Role.AtLeast(rbac.RoleContributor) {
		return domain.ErrForbidden
	}

	problem, err := uc.problemRepo.GetByID(ctx, params.ID)
	if err != nil {
		if errors.Is(err, domain.ErrProblemNotFound) {
			return domain.ErrProblemNotFound
		}
		return domain.ErrInternalServer.Wrap(err)
	}

	if claims.Role == rbac.RoleContributor {
		if problem.AuthorID != claims.UserID {
			return domain.ErrForbidden
		}
		if !problem.IsHidden {
			return domain.ErrForbidden
		}
	}

	tc, err := uc.testcaseRepo.GetByProblemID(ctx, problem.ID)
	if err != nil {
		if errors.Is(err, domain.ErrTestCaseNotFound) {
			return domain.ErrTestCaseNotFound
		}
		return domain.ErrInternalServer.Wrap(err)
	}

	if err := uc.testcaseStorage.DeleteTestCase(ctx, tc.ZipObjectKey); err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	if err := uc.testcaseRepo.DeleteByProblemID(ctx, problem.ID); err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	return nil
}
