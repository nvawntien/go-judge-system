package usecase

import (
	"context"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"

	"go.uber.org/zap"
)

type verifyActivationUseCase struct {
	otpService outbound.OTPService
	userRepo   outbound.UserRepository
	logger     *zap.Logger
}

func NewVerifyActivationUseCase(otpService outbound.OTPService, userRepo outbound.UserRepository, logger *zap.Logger) inbound.VerifyActivationUseCase {
	return &verifyActivationUseCase{
		otpService: otpService,
		userRepo:   userRepo,
		logger:     logger,
	}
}

func (uc *verifyActivationUseCase) Execute(ctx context.Context, req dto.VerifyOTPRequest) error {
	if err := uc.otpService.VerifyOTP(ctx, "activation", req.Email, req.OTP); err != nil {
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

	if user.IsActive {
		return domain.ErrUserAlreadyActive
	}

	user.Activate()

	if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
		uc.logger.Error("failed to update user status to active", zap.String("email", req.Email), zap.Error(err))
		return domain.ErrInternalServer
	}

	uc.otpService.Cleanup(ctx, "activation", req.Email)

	uc.logger.Info("User account activated successfully",
		zap.String("email", req.Email),
	)
	return nil
}
