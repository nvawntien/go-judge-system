package inbound

import (
	"context"
	"go-judge-system/services/auth/internal/application/dto"
)

type AssignRoleUseCase interface {
	Execute(ctx context.Context, params dto.UserIDRequest, body dto.AssignRoleRequest) error
}
