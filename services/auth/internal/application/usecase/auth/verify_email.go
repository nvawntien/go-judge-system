package auth

import (
	"context"
	"errors"

	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"

	"go.uber.org/zap"
)

type verifyEmail struct {
	tokenGenerator outbound.TokenGenerator
	tokenRepo      outbound.TokenRepository
	userRepo       outbound.UserRepository
	logger         *zap.Logger
}

func NewVerifyEmailUseCase(
	tokenGenerator outbound.TokenGenerator,
	tokenRepo outbound.TokenRepository,
	userRepo outbound.UserRepository,
	logger *zap.Logger,
) inbound.VerifyEmailUseCase {
	return &verifyEmail{
		tokenGenerator: tokenGenerator,
		tokenRepo:      tokenRepo,
		userRepo:       userRepo,
		logger:         logger,
	}
}

func (uc *verifyEmail) Execute(ctx context.Context, req dto.VerifyEmailRequest) error {
	// Hash the raw token to look up in Redis
	hashedToken := uc.tokenGenerator.Hash(req.Token)

	// Find the associated email
	userID, err := uc.tokenRepo.FindByToken(ctx, hashedToken)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidOrExpiredToken) {
			return domain.ErrInvalidOrExpiredToken
		}
		uc.logger.Error("failed to look up verification token", zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	// Find the user
	user, err := uc.userRepo.GetUserById(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.ErrUserNotFound
		}
		uc.logger.Error("failed to get user by ID", zap.String("user_id", userID), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	// Check if already active
	if user.IsActive {
		return domain.ErrUserAlreadyActive
	}

	// Activate user
	user.Activate()
	if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
		uc.logger.Error("failed to activate user", zap.String("user_id", userID), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	// Clean up token
	if err := uc.tokenRepo.Delete(ctx, hashedToken); err != nil {
		uc.logger.Warn("failed to delete verification token after activation", zap.String("user_id", userID), zap.Error(err))
	}

	uc.logger.Info("user email verified successfully", zap.String("user_id", userID))
	return nil
}
