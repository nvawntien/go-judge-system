package inbound

import (
	"context"
	pkgAuth "go-judge-system/pkg/auth"
	"go-judge-system/services/auth/internal/application/dto"
)

type GetMeUseCase interface {
	Execute(ctx context.Context, claims pkgAuth.Claims) (*dto.GetMeResponse, error)
}

type GetProfileUseCase interface {
	Execute(ctx context.Context, req dto.GetProfileRequest) (dto.GetProfileResponse, error)
}
