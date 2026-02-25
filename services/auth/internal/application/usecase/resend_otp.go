package usecase

import (
	"context"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"

	"go.uber.org/zap"
)

type resendOTPUseCase struct {
	userRepo outbound.UserRepository
	otpUC    inbound.OTPUseCase
	logger   *zap.Logger
}

func NewResendOTPUseCase(userRepo outbound.UserRepository, otpUC inbound.OTPUseCase, logger *zap.Logger) inbound.ResendOTPUseCase {
	return &resendOTPUseCase{
		userRepo: userRepo,
		otpUC:    otpUC,
		logger:   logger,
	}
}

func (uc *resendOTPUseCase) Execute(ctx context.Context, req dto.ResendOTPRequest) error {
	user, err := uc.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if err != domain.ErrUserNotFound {
			uc.logger.Error("failed to retrieve user for OTP resend", zap.String("email", req.Email), zap.Error(err))
			return domain.ErrInternalServer
		}
		return err
	}

	switch req.Purpose {
		case "activation":
			if user.IsActive {
				return domain.ErrUserAlreadyActive
			}
		case "forgot_password":
			if !user.IsActive {
				return domain.ErrUserInactive
			}
	}

	if err := uc.otpUC.RequestOTP(ctx, req.Purpose, req.Email); err != nil {
		uc.logger.Error("failed to request OTP for resend", zap.String("email", req.Email), zap.Error(err))
		return err
	}	

	uc.logger.Info("OTP resent successfully",
		zap.String("email", req.Email),
		zap.String("purpose", req.Purpose),
	)
	return nil
}
