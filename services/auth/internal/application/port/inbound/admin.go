package inbound

import (
	"context"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/auth/internal/application/dto"
)

type AssignRoleUseCase interface {
	Execute(ctx context.Context, params dto.UserIDRequest, body dto.AssignRoleRequest) error
}

type AdminUsersUseCase interface {
	List(ctx context.Context, claims auth.Claims, req dto.ListAdminUsersRequest) (dto.ListAdminUsersResponse, error)
	Get(ctx context.Context, claims auth.Claims, params dto.UserIDRequest) (dto.AdminUserResponse, error)
	SetSuspension(ctx context.Context, claims auth.Claims, params dto.UserIDRequest, req dto.SetUserSuspensionRequest) (dto.AdminUserResponse, error)
}
