package admin
import (
	"context"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/problem/internal/application/dto"
)

type CreateProblemUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, req dto.CreateProblemRequest) (dto.ProblemDetailResponse, error)
}

type PublishProblemUseCase interface {
	Execute(ctx context.Context, params dto.ProblemIDRequest) (dto.ProblemDetailResponse, error)
}

type HiddenProblemUseCase interface {
	Execute(ctx context.Context, params dto.ProblemIDRequest) (dto.ProblemDetailResponse, error)
}