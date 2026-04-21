package auth

import (
	"context"
	"errors"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/valueobject"
	"time"

	"go.uber.org/zap"
)

const (
    forgotPasswordTokenTTL    = 5 * time.Minute
    forgotPasswordCooldownTTL = 60 * time.Second
)

type forgotPasswordUseCase struct {
	userRepo       outbound.UserRepository
	tokenRepo      outbound.TokenRepository
	tokenGenerator outbound.TokenGenerator
	mailProvider   outbound.MailProvider
	logger         *zap.Logger
}

func NewForgotPasswordUseCase(
	userRepo outbound.UserRepository,
	tokenRepo outbound.TokenRepository,
	tokenGenerator outbound.TokenGenerator,
	mailProvider outbound.MailProvider,
	logger *zap.Logger,
) inbound.ForgotPasswordUseCase {
	return &forgotPasswordUseCase{
		userRepo:       userRepo,
		tokenRepo:      tokenRepo,
		tokenGenerator: tokenGenerator,
		mailProvider:   mailProvider,
		logger:         logger,
	}
}

func (uc *forgotPasswordUseCase) Execute(ctx context.Context, req dto.ForgotPasswordRequest) error {
	emailVO, err := valueobject.NewEmail(req.Email)
	if err != nil {
		return err
	}

	email := emailVO.String()
	
	user, err := uc.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil
		}
		uc.logger.Error("failed to get user by email for forgot password", zap.String("email", email), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	allowed, err := uc.tokenRepo.TryAcquireResendCooldown(ctx, user.ID, forgotPasswordCooldownTTL)
	if err != nil {
		uc.logger.Error("failed to apply resend verification cooldown", zap.String("user_id", user.ID), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	if !allowed {
		return domain.ErrRateLimitExceeded
	}

	rawToken := uc.tokenGenerator.Generate(user.ID)
	hashedToken := uc.tokenGenerator.Hash(rawToken)

	if err := uc.tokenRepo.Save(ctx, hashedToken, user.ID, forgotPasswordTokenTTL); err != nil {
		uc.logger.Error("failed to save forgot password token", zap.String("user_id", user.ID), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	if err := uc.mailProvider.SendForgotPasswordEmail(ctx, user.Email, rawToken); err != nil {
		uc.logger.Error("failed to send forgot password email", zap.String("user_id", user.ID), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	uc.logger.Info("sent forgot password email", zap.String("user_id", user.ID))
	return nil
}
