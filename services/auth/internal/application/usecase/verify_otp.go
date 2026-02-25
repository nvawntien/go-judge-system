package usecase

import (
	"context"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"

	"go.uber.org/zap"
)

type verifyOTPUseCase struct {
	otpUC    inbound.OTPUseCase
	userRepo outbound.UserRepository
	logger   *zap.Logger
}

func NewVerifyOTPUseCase(otpUC inbound.OTPUseCase, userRepo outbound.UserRepository, logger *zap.Logger) inbound.VerifyOTPUseCase {
	return &verifyOTPUseCase{
		otpUC:    otpUC,
		userRepo: userRepo,
		logger:   logger,
	}
}

func (uc *verifyOTPUseCase) Execute(ctx context.Context, req dto.VerifyOTPRequest) error {
	if err := uc.otpUC.VerifyOTP(ctx, req.Purpose, req.Email, req.OTP); err != nil {
		uc.logger.Warn("OTP verification failed",
			zap.String("email", req.Email),
			zap.Error(err),
		)
		return err
	}

	user, err := uc.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if err != domain.ErrUserNotFound {
			uc.logger.Error("failed to retrieve user after OTP verification", zap.String("email", req.Email), zap.Error(err))
			return domain.ErrInternalServer
		}
		return err
	}

	switch req.Purpose {
		case "activation":
			if user.IsActive {
				return domain.ErrUserAlreadyActive
			}
			user.IsActive = true
			if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
				uc.logger.Error("Failed to activate user",
					zap.String("email", req.Email),
					zap.Error(err),
				)
				return domain.ErrInternalServer
			}
		case "forgot_password":
			// For forgot password, we might want to generate a password reset token instead of activating the user
			// This is just a placeholder for the actual implementation
			uc.logger.Info("Password reset OTP verified",
				zap.String("email", req.Email),
			)
	}			
	

	uc.otpUC.Cleanup(ctx, req.Purpose, req.Email)

	uc.logger.Info("User verified successfully",
		zap.String("email", req.Email),
	)
	return nil
}
