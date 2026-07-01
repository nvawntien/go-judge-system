package admin
import (
	"context"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/problem/internal/application/dto"
)

// CreateProblemUseCase: HandleWithClaims → fn(ctx, claims, Req) (Res, err)
type CreateProblemUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, req dto.CreateProblemRequest) (dto.ProblemDetailResponse, error)
}