package problem

import (
	"context"
	"strings"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/pkg/response"
	"go-judge-system/services/problem/internal/application/dto"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/application/usecase"
	"go-judge-system/services/problem/internal/domain"
	"go-judge-system/services/problem/internal/domain/entity"
)

type listProblemsUseCase struct {
	problemRepo outbound.ProblemRepository
}

func NewListProblemsUseCase(problemRepo outbound.ProblemRepository) inbound.ListProblemsUseCase {
	return &listProblemsUseCase{problemRepo: problemRepo}
}

func (uc *listProblemsUseCase) Execute(ctx context.Context, claims auth.Claims, req dto.ListProblemsRequest) (dto.ListProblemsResponse, error) {
	if !claims.Role.AtLeast(rbac.RoleModerator) {
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
	search := strings.TrimSpace(req.Search)
	tagSlug := strings.ToLower(strings.TrimSpace(req.TagSlug))

	if difficulty != "" {
		switch difficulty {
		case string(entity.Easy), string(entity.Medium), string(entity.Hard):
		default:
			return dto.ListProblemsResponse{}, response.NewAppError(response.CodeBadRequest, "difficulty must be one of easy, medium, hard", nil)
		}
	}

	offset := (page - 1) * limit

	problems, err := uc.problemRepo.List(ctx, offset, limit, difficulty, search, tagSlug, true)
	if err != nil {
		return dto.ListProblemsResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	total, err := uc.problemRepo.Count(ctx, difficulty, search, tagSlug, true)
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
