package auth

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

type resetPasswordUseCase struct {
	userRepo        outbound.UserRepository
	tokenRepo       outbound.TokenRepository
	tokenGenerator  outbound.TokenGenerator
	passwordEncoder outbound.PasswordEncoder
	logger          *zap.Logger
}

func NewResetPasswordUseCase(
	userRepo outbound.UserRepository,
	tokenRepo outbound.TokenRepository,
	tokenGenerator outbound.TokenGenerator,
	passwordEncoder outbound.PasswordEncoder,
	logger *zap.Logger,
) inbound.ResetPasswordUseCase {
	return &resetPasswordUseCase{
		userRepo:        userRepo,
		tokenRepo:       tokenRepo,
		tokenGenerator:  tokenGenerator,
		passwordEncoder: passwordEncoder,
		logger:          logger,
	}
}

func (uc *resetPasswordUseCase) Execute(ctx context.Context, req dto.ResetPasswordRequest) error {
	// Hash the raw token to look up in Redis
	if err := valueobject.ValidatePlainPassword(req.NewPassword); err != nil {
		return err
	}

	if req.NewPassword != req.ConfirmPassword {
		return domain.ErrPasswordMismatch
	}

	hashedToken := uc.tokenGenerator.Hash(req.Token)
	// Find the associated user ID
	userID, err := uc.tokenRepo.FindByToken(ctx, hashedToken)
	if err != nil {
		uc.logger.Error("failed to look up reset password token", zap.Error(err))
		return domain.ErrInvalidOrExpiredToken
	}

	// Find the user
	user, err := uc.userRepo.GetUserById(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.ErrUserNotFound
		}
		uc.logger.Error("failed to get user by ID for reset password", zap.String("user_id", userID), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	// Encode the new password
	hashedPassword, err := uc.passwordEncoder.HashAndSalt([]byte(req.NewPassword))
	if err != nil {
		uc.logger.Error("failed to encode new password", zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	passwordVO := valueobject.NewPasswordFromHash(hashedPassword)

	user.UpdatePassword(passwordVO)
	// Update the user's password
	if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
		uc.logger.Error("failed to update user password", zap.String("user_id", user.ID), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	// Clean up token
	if err := uc.tokenRepo.Delete(ctx, hashedToken); err != nil {
		uc.logger.Warn("failed to delete verification token after activation", zap.String("user_id", userID), zap.Error(err))
	}

	uc.logger.Info("password reset successful", zap.String("user_id", user.ID))

	return nil
}
