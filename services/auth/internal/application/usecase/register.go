package usecase

import (
	"context"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"
	"go-judge-system/services/auth/internal/domain/valueobject"

	"go.uber.org/zap"
)

type registerUseCase struct {
	userRepo outbound.UserRepository
	otpUC    inbound.OTPUseCase
	logger   *zap.Logger
}

func NewRegisterUseCase(userRepo outbound.UserRepository, otpUC inbound.OTPUseCase, logger *zap.Logger) inbound.RegisterUseCase {
	return &registerUseCase{
		userRepo: userRepo,
		otpUC:    otpUC,
		logger:   logger,
	}
}

func (uc *registerUseCase) Execute(ctx context.Context, req dto.RegisterRequest) error {
	exists, err := uc.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil && err != domain.ErrUserNotFound {
		uc.logger.Error("failed to check existing user", zap.String("email", req.Email), zap.Error(err))
		return domain.ErrInternalServer
	}

	if exists != nil {
		return domain.ErrUserAlreadyExists
	}

	emailVO, err := valueobject.NewEmail(req.Email)
	if err != nil {
		uc.logger.Error("failed to create email value object", zap.String("email", req.Email), zap.Error(err))
		return domain.ErrInvalidEmail
	}

	passwordVO, err := valueobject.NewPasswordFromPlain(req.Password)
	if err != nil {
		uc.logger.Error("failed to create password value object", zap.String("email", req.Email), zap.Error(err))
		return err
	}

	user := entity.NewUser(req.Username, emailVO, passwordVO)
	if err := uc.userRepo.CreateUser(ctx, user); err != nil {
		uc.logger.Error("failed to create user", zap.String("email", req.Email), zap.Error(err))
		return domain.ErrInternalServer
	}

	if err := uc.otpUC.RequestOTP(ctx, "activation", req.Email); err != nil {
		uc.logger.Error("failed to request OTP after registration", zap.String("email", req.Email), zap.Error(err))
		return err
	}

	return nil
}
