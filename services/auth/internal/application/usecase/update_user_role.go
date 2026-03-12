package usecase

import (
	"context"
	"errors"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"

	"go.uber.org/zap"
)

type updateUserRoleUseCase struct {
	userRepo outbound.UserRepository
	logger   *zap.Logger
}

func NewUpdateUserRoleUseCase(userRepo outbound.UserRepository, logger *zap.Logger) inbound.UpdateUserRoleUseCase {
	return &updateUserRoleUseCase{
		userRepo: userRepo,
		logger:   logger,
	}
}

func (uc *updateUserRoleUseCase) Execute(ctx context.Context, claims auth.Claims, params dto.UserRoleRequest, body dto.UpdateUserRoleRequest) error {
	if !claims.IsSuperAdmin() {
		return domain.ErrForbidden
	}

	if body.Role == nil {
		return nil // nothing to change
	}

	user, err := uc.userRepo.GetUserByUsername(ctx, params.Username)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.ErrUserNotFound
		}
		uc.logger.Error("failed to get user", zap.Error(err), zap.String("username", params.Username))
		return domain.ErrInternalServer.Wrap(err)
	}

	if user.Role == *body.Role {
		return nil
	}

	user.AssignRole(*body.Role)

	if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
		uc.logger.Error("failed to update user role", zap.Error(err), zap.String("username", params.Username))
		return domain.ErrInternalServer.Wrap(err)
	}

	return nil
}
