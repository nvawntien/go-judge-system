package usecase

import (
	"context"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/valueobject"

	"go.uber.org/zap"
)

type resetPasswordUseCase struct {
	userRepo  outbound.UserRepository
	tokenRepo outbound.ResetTokenRepository
	tokenGen  outbound.TokenGenerator
	logger    *zap.Logger
}

func NewResetPasswordUseCase(userRepo outbound.UserRepository, tokenRepo outbound.ResetTokenRepository, tokenGen outbound.TokenGenerator, logger *zap.Logger) inbound.ResetPasswordUseCase {
	return &resetPasswordUseCase{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		tokenGen:  tokenGen,
		logger:    logger,
	}
}

func (uc *resetPasswordUseCase) Execute(ctx context.Context, req dto.ResetPasswordRequest) error {
	hashedToken := uc.tokenGen.HashToken(req.ResetToken)

	email, err := uc.tokenRepo.Get(ctx, hashedToken)
	if err != nil {
		uc.logger.Warn("failed to get email from reset token", zap.String("reset_token", req.ResetToken), zap.Error(err))
		return domain.ErrInvalidOrExpiredToken
	}

	passwordVO, err := valueobject.NewPasswordFromPlain(req.NewPassword)
	if err != nil {
		uc.logger.Error("invalid new password format", zap.String("email", email), zap.Error(err))
		return err
	}

	user, err := uc.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		uc.logger.Error("failed to get user by email", zap.String("email", email), zap.Error(err))
		return err
	}

	user.Password = passwordVO.Hash()
	err = uc.userRepo.UpdateUser(ctx, user)
	if err != nil {
		uc.logger.Error("failed to update user password", zap.String("email", email), zap.Error(err))
		return err
	}

	if err := uc.tokenRepo.Delete(ctx, hashedToken); err != nil {
		uc.logger.Warn("failed to delete reset token", zap.String("reset_token", req.ResetToken), zap.Error(err))
	}

	uc.logger.Info("Password reset successfully",
		zap.String("email", email),
	)
	return nil
}