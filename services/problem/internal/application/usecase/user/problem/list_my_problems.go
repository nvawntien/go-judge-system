package problem

import (
	"context"
	"strings"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/problem/internal/application/dto"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/user"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/application/usecase"
	"go-judge-system/services/problem/internal/domain"
)

type listMyProblemsUseCase struct {
	problemRepo outbound.ProblemRepository
}

func NewListMyProblemsUseCase(problemRepo outbound.ProblemRepository) inbound.ListMyProblemsUseCase {
	return &listMyProblemsUseCase{problemRepo: problemRepo}
}

func (uc *listMyProblemsUseCase) Execute(ctx context.Context, claims auth.Claims, req dto.ListProblemsRequest) (dto.ListProblemsResponse, error) {
	if !claims.Role.AtLeast(rbac.RoleContributor) {
		return dto.ListProblemsResponse{}, domain.ErrForbidden
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	difficulty := strings.ToLower(strings.TrimSpace(req.Difficulty))
	if difficulty != "" && !isValidDifficulty(difficulty) {
		return dto.ListProblemsResponse{}, domain.ErrInvalidDifficulty
	}

	search := strings.TrimSpace(req.Search)
	tagSlug := strings.ToLower(strings.TrimSpace(req.TagSlug))
	offset := (page - 1) * limit

	problems, err := uc.problemRepo.ListByAuthor(ctx, claims.UserID, offset, limit, difficulty, search, tagSlug)
	if err != nil {
		return dto.ListProblemsResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	total, err := uc.problemRepo.CountByAuthor(ctx, claims.UserID, difficulty, search, tagSlug)
	if err != nil {
		return dto.ListProblemsResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	items := make([]dto.ProblemResponse, 0, len(problems))
	for _, problem := range problems {
		items = append(items, usecase.MapProblemToResponse(problem, true))
	}

	return dto.ListProblemsResponse{
		Items: items,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}
