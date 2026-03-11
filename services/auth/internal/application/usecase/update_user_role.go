package usecase

import (
	"context"

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

func (uc *updateUserRoleUseCase) Execute(ctx context.Context, claims auth.Claims, req dto.UpdateUserRoleRequest) error {
	if !claims.IsSuperAdmin() {
		return domain.ErrForbidden
	}

	if req.Role == nil {
		return nil // nothing to change
	}

	user, err := uc.userRepo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		uc.logger.Error("failed to get user", zap.Error(err), zap.String("username", req.Username))
		return domain.ErrInternalServer.Wrap(err)
	}

	if user.Role == *req.Role {
		return nil
	}

	user.AssignRole(*req.Role)

	if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
		uc.logger.Error("failed to update user role", zap.Error(err), zap.String("username", req.Username))
		return domain.ErrInternalServer.Wrap(err)
	}

	return nil
}
