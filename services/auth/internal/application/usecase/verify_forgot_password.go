package usecase

import (
	"context"
	"errors"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"time"

	"go.uber.org/zap"
)

type verifyForgotPasswordUseCase struct {
	otpService     outbound.OTPService
	userRepository outbound.UserRepository
	resetTokenRepo outbound.ResetTokenRepository
	tokenGen       outbound.ResetTokenGenerator
	logger         *zap.Logger
}

func NewVerifyForgotPasswordUseCase(otpService outbound.OTPService, userRepository outbound.UserRepository, resetTokenRepo outbound.ResetTokenRepository, tokenGen outbound.ResetTokenGenerator, logger *zap.Logger) inbound.VerifyForgotPasswordUseCase {
	return &verifyForgotPasswordUseCase{
		otpService:     otpService,
		userRepository: userRepository,
		resetTokenRepo: resetTokenRepo,
		tokenGen:       tokenGen,
		logger:         logger,
	}
}

func (uc *verifyForgotPasswordUseCase) Execute(ctx context.Context, req dto.VerifyOTPRequest) (string, error) {
	if err := uc.otpService.VerifyOTP(ctx, "forgot_password", req.Email, req.OTP); err != nil {
		return "", err
	}

	user, err := uc.userRepository.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if !errors.Is(err, domain.ErrUserNotFound) {
			uc.logger.Error("failed to retrieve user", zap.String("email", req.Email), zap.Error(err))
			return "", domain.ErrInternalServer.Wrap(err)
		}
		return "", domain.ErrUserNotFound
	}

	if !user.IsActive {
		return "", domain.ErrUserInactive
	}

	rawToken := uc.tokenGen.Generate(user.ID)
	hashedToken := uc.tokenGen.Hash(rawToken)

	if err := uc.resetTokenRepo.Save(ctx, hashedToken, req.Email, 15*time.Minute); err != nil {
		uc.logger.Error("failed to save reset token", zap.String("email", req.Email), zap.Error(err))
		return "", domain.ErrInternalServer.Wrap(err)
	}

	uc.otpService.Cleanup(ctx, "forgot_password", req.Email)

	uc.logger.Info("Forgot password OTP verified successfully, reset token generated",
		zap.String("email", req.Email),
	)
	return rawToken, nil
}
