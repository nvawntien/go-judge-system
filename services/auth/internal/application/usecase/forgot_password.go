package usecase

import (
	"context"
	"errors"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"

	"go.uber.org/zap"
)

type forgotPasswordUseCase struct {
	userRepo   outbound.UserRepository
	otpService outbound.OTPService
	logger     *zap.Logger
}

func NewForgotPasswordUseCase(userRepo outbound.UserRepository, otpService outbound.OTPService, logger *zap.Logger) inbound.ForgotPasswordUseCase {
	return &forgotPasswordUseCase{
		userRepo:   userRepo,
		otpService: otpService,
		logger:     logger,
	}
}

func (uc *forgotPasswordUseCase) Execute(ctx context.Context, req dto.ForgotPasswordRequest) error {
	user, err := uc.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if !errors.Is(err, domain.ErrUserNotFound) {
			uc.logger.Error("failed to retrieve user for forgot password", zap.String("email", req.Email), zap.Error(err))
			return domain.ErrInternalServer.Wrap(err)
		}
		return domain.ErrUserNotFound
	}

	if !user.IsActive {
		return domain.ErrUserInactive
	}

	if err := uc.otpService.RequestOTP(ctx, "forgot_password", req.Email); err != nil {
		uc.logger.Error("failed to request OTP for forgot password", zap.String("email", req.Email), zap.Error(err))
		return err
	}

	uc.logger.Info("Forgot password OTP requested successfully",
		zap.String("email", req.Email),
	)
	return nil
}
