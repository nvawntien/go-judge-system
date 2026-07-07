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

type getProblemUseCase struct {
	problemRepo  outbound.ProblemRepository
	testcaseRepo outbound.TestCaseRepository
}

func NewGetProblemUseCase(problemRepo outbound.ProblemRepository, testcaseRepo outbound.TestCaseRepository) inbound.GetProblemUseCase {
	return &getProblemUseCase{
		problemRepo:  problemRepo,
		testcaseRepo: testcaseRepo,
	}
}

func (uc *getProblemUseCase) Execute(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest) (dto.AdminProblemDetailResponse, error) {
	if !claims.Role.AtLeast(rbac.RoleContributor) {
		return dto.AdminProblemDetailResponse{}, domain.ErrForbidden
	}

	problem, err := uc.problemRepo.GetByID(ctx, params.ID)
	if err != nil {
		if errors.Is(err, domain.ErrProblemNotFound) {
			return dto.AdminProblemDetailResponse{}, domain.ErrProblemNotFound
		}
		return dto.AdminProblemDetailResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	if !claims.Role.AtLeast(rbac.RoleModerator) && problem.AuthorID != claims.UserID {
		return dto.AdminProblemDetailResponse{}, domain.ErrForbidden
	}

	tc, err := uc.testcaseRepo.GetByProblemID(ctx, problem.ID)
	if err != nil {
		if !errors.Is(err, domain.ErrTestCaseNotFound) {
			return dto.AdminProblemDetailResponse{}, domain.ErrInternalServer.Wrap(err)
		}
		tc = nil
	}

	return usecase.MapProblemToAdminDetailResponse(problem, tc), nil
}
