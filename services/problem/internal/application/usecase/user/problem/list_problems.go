package problem

import (
	"context"
	"strings"

	"go-judge-system/services/problem/internal/application/dto"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/user"
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

func (uc *listProblemsUseCase) Execute(ctx context.Context, req dto.ListProblemsRequest) (dto.ListProblemsResponse, error) {
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
	offset := (page - 1) * limit

	problems, err := uc.problemRepo.List(ctx, offset, limit, difficulty, search, false)
	if err != nil {
		return dto.ListProblemsResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	total, err := uc.problemRepo.Count(ctx, difficulty, search, false)
	if err != nil {
		return dto.ListProblemsResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	items := make([]dto.ProblemResponse, 0, len(problems))
	for _, problem := range problems {
		items = append(items, usecase.MapProblemToResponse(problem, false))
	}

	return dto.ListProblemsResponse{
		Items: items,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func isValidDifficulty(difficulty string) bool {
	switch difficulty {
	case string(entity.Easy), string(entity.Medium), string(entity.Hard):
		return true
	default:
		return false
	}
}
