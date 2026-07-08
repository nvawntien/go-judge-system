package testcase

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

type getTestCaseUseCase struct {
	problemRepo  outbound.ProblemRepository
	testcaseRepo outbound.TestCaseRepository
}

func NewGetTestCaseUseCase(problemRepo outbound.ProblemRepository, testcaseRepo outbound.TestCaseRepository) inbound.GetTestCaseUseCase {
	return &getTestCaseUseCase{
		problemRepo:  problemRepo,
		testcaseRepo: testcaseRepo,
	}
}

func (uc *getTestCaseUseCase) Execute(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest) (dto.TestCaseMetadataResponse, error) {
	if !claims.Role.AtLeast(rbac.RoleContributor) {
		return dto.TestCaseMetadataResponse{}, domain.ErrForbidden
	}

	problem, err := uc.problemRepo.GetByID(ctx, params.ID)
	if err != nil {
		if errors.Is(err, domain.ErrProblemNotFound) {
			return dto.TestCaseMetadataResponse{}, domain.ErrProblemNotFound
		}
		return dto.TestCaseMetadataResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	if !claims.Role.AtLeast(rbac.RoleModerator) && problem.AuthorID != claims.UserID {
		return dto.TestCaseMetadataResponse{}, domain.ErrForbidden
	}

	tc, err := uc.testcaseRepo.GetByProblemID(ctx, problem.ID)
	if err != nil {
		if errors.Is(err, domain.ErrTestCaseNotFound) {
			return dto.TestCaseMetadataResponse{}, domain.ErrTestCaseNotFound
		}
		return dto.TestCaseMetadataResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	return usecase.MapTestCaseToMetadataResponse(tc), nil
}
