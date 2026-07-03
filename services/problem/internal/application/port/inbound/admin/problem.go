package admin

import (
	"context"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/problem/internal/application/dto"
)

type CreateProblemUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, req dto.CreateProblemRequest) (dto.ProblemDetailResponse, error)
}

type ListProblemsUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, req dto.ListProblemsRequest) (dto.ListProblemsResponse, error)
}

type GetProblemUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest) (dto.AdminProblemDetailResponse, error)
}

type PublishProblemUseCase interface {
	Execute(ctx context.Context, params dto.ProblemIDRequest) (dto.ProblemDetailResponse, error)
}

type HiddenProblemUseCase interface {
	Execute(ctx context.Context, params dto.ProblemIDRequest) (dto.ProblemDetailResponse, error)
}
