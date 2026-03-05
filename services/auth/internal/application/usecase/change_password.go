package usecase

import (
	"context"
	"errors"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/valueobject"

	"go.uber.org/zap"
)

type changePasswordUseCase struct {
	userRepo       outbound.UserRepository
	passwordHasher outbound.PasswordHasher
	logger         *zap.Logger
}

func NewChangePasswordUseCase(userRepo outbound.UserRepository, passwordHasher outbound.PasswordHasher, logger *zap.Logger) inbound.ChangePasswordUseCase {
	return &changePasswordUseCase{
		userRepo:       userRepo,
		passwordHasher: passwordHasher,
		logger:         logger,
	}
}

func (uc *changePasswordUseCase) Execute(ctx context.Context, userID string, req dto.ChangePasswordRequest) error {
	user, err := uc.userRepo.GetUserById(ctx, userID)
	if err != nil {
		if !errors.Is(err, domain.ErrUserNotFound) {
			uc.logger.Error("failed to get user by id", zap.String("user_id", userID), zap.Error(err))
			return domain.ErrInternalServer.Wrap(err)
		}
		return domain.ErrUserNotFound
	}

	match, err := uc.passwordHasher.Compare(user.Password, req.OldPassword)
	if err != nil {
		uc.logger.Error("failed to compare password", zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}
	if !match {
		return domain.ErrIncorrecOldPassword
	}

	if err := valueobject.ValidatePlainPassword(req.NewPassword); err != nil {
		return err
	}

	hashedPassword, err := uc.passwordHasher.Hash(req.NewPassword)
	if err != nil {
		uc.logger.Error("failed to hash new password", zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	passwordVO := valueobject.NewPasswordFromHash(hashedPassword)
	user.UpdatePassword(passwordVO)

	if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
		uc.logger.Error("failed to update user password", zap.String("user_id", userID), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	return nil
}
