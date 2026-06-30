package admin

import (
	"context"
	"errors"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
)

type assignRoleUseCase struct {
	userRepo outbound.UserRepository
}

func NewAssignRoleUseCase(userRepo outbound.UserRepository) inbound.AssignRoleUseCase {
	return &assignRoleUseCase{userRepo: userRepo}
}

func (uc *assignRoleUseCase) Execute(ctx context.Context, params dto.UserIDRequest, body dto.AssignRoleRequest) error {
	user, err := uc.userRepo.GetUserById(ctx, params.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.ErrUserNotFound
		}
		return domain.ErrInternalServer.Wrap(err)
	}

	if user.Role == body.Role {
		return domain.ErrAssignRoleAlreadyAssigned
	}	

	user.AssignRole(body.Role)
	if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	return nil
}
