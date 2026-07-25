package admin

import (
	"context"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/submission/internal/application/dto"
)

type ListAdminSubmissionsUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, req dto.ListAdminSubmissionsRequest) (dto.ListAdminSubmissionsResponse, error)
}
