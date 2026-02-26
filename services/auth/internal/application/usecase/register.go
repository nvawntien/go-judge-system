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
	userRepo       outbound.UserRepository
	passwordHasher outbound.PasswordHasher
	otpUC          inbound.OTPUseCase
	logger         *zap.Logger
}

func NewRegisterUseCase(
	userRepo outbound.UserRepository,
	passwordHasher outbound.PasswordHasher,
	otpUC inbound.OTPUseCase,
	logger *zap.Logger,
) inbound.RegisterUseCase {
	return &registerUseCase{
		userRepo:       userRepo,
		passwordHasher: passwordHasher,
		otpUC:          otpUC,
		logger:         logger,
	}
}

func (uc *registerUseCase) Execute(ctx context.Context, req dto.RegisterRequest) error {
	existingUser, err := uc.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil && err != domain.ErrUserNotFound {
		uc.logger.Error(
			"failed to check existing user",
			zap.String("email", req.Email),
			zap.Error(err),
		)
		return domain.ErrInternalServer
	}

	if existingUser != nil {
		return domain.ErrUserAlreadyExists
	}

	emailVO, err := valueobject.NewEmail(req.Email)
	if err != nil {
		return domain.ErrInvalidEmail
	}

	if err := valueobject.ValidatePlainPassword(req.Password); err != nil {
		return err
	}

	hashedPassword, err := uc.passwordHasher.Hash(req.Password)
	if err != nil {
		uc.logger.Error(
			"failed to hash password",
			zap.String("email", req.Email),
			zap.Error(err),
		)
		return domain.ErrInternalServer
	}

	passwordVO := valueobject.NewPasswordFromHash(hashedPassword)

	user := entity.NewUser(
		req.Username,
		emailVO,
		passwordVO,
	)

	if err := uc.userRepo.CreateUser(ctx, user); err != nil {
		uc.logger.Error(
			"failed to create user",
			zap.String("email", req.Email),
			zap.Error(err),
		)
		return domain.ErrInternalServer
	}

	if err := uc.otpUC.RequestOTP(ctx, "activation", req.Email); err != nil {
		uc.logger.Error(
			"failed to request OTP after registration",
			zap.String("email", req.Email),
			zap.Error(err),
		)
		return err
	}

	return nil
}