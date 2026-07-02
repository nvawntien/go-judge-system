package user

import (
	"context"

	"go-judge-system/services/problem/internal/application/dto"
)

type ListProblemsUseCase interface {
	Execute(ctx context.Context, req dto.ListProblemsRequest) (dto.ListProblemsResponse, error)
}

type GetProblemUseCase interface {
	Execute(ctx context.Context, params dto.ProblemSlugRequest) (dto.ProblemDetailResponse, error)
}
