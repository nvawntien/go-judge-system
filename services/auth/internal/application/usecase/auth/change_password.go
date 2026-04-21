package auth

import (
	"context"
	"errors"
	"go-judge-system/pkg/auth"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/valueobject"

	"go.uber.org/zap"
)

type changePasswordUseCase struct {
	userRepo        outbound.UserRepository
	passwordEncoder outbound.PasswordEncoder
	logger          *zap.Logger
}

func NewChangePasswordUseCase(
	userRepo outbound.UserRepository,
	passwordEncoder outbound.PasswordEncoder,
	logger *zap.Logger,
) inbound.ChangePasswordUseCase {
	return &changePasswordUseCase{
		userRepo:        userRepo,
		passwordEncoder: passwordEncoder,
		logger:          logger,
	}
}

func (uc *changePasswordUseCase) Execute(ctx context.Context, claims auth.Claims, req dto.ChangePasswordRequest) error {
	if err := valueobject.ValidatePlainPassword(req.NewPassword); err != nil {
		return err
	}

	userID := claims.UserID

	// Find the user
	user, err := uc.userRepo.GetUserById(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.ErrUserNotFound
		}
		uc.logger.Error("failed to get user by ID for change password", zap.String("user_id", userID), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	// Check if the current password is correct
	if check := uc.passwordEncoder.ComparePasswords(user.Password, []byte(req.CurrentPassword)); !check {
		return domain.ErrIncorrectCurrentPassword
	}

	if req.CurrentPassword == req.NewPassword {
		return domain.ErrNewPasswordSameAsCurrent
	}

	if req.NewPassword != req.ConfirmPassword {
		return domain.ErrPasswordMismatch
	}

	// Hash the new password
	hashedPassword, err := uc.passwordEncoder.HashAndSalt([]byte(req.NewPassword))
	if err != nil {
		uc.logger.Error("failed to hash new password for change password", zap.String("user_id", userID), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	passwordVO := valueobject.NewPasswordFromHash(hashedPassword)

	// Update the user's password
	user.UpdatePassword(passwordVO)
	if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
		uc.logger.Error("failed to update user password", zap.String("user_id", userID), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	uc.logger.Info("user password changed successfully", zap.String("user_id", userID))
	return nil
}
