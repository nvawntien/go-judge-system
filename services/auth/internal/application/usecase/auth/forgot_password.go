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
}

func NewForgotPasswordUseCase(
	userRepo outbound.UserRepository,
	tokenRepo outbound.TokenRepository,
	tokenGenerator outbound.TokenGenerator,
	mailProvider outbound.MailProvider,
) inbound.ForgotPasswordUseCase {
	return &forgotPasswordUseCase{
		userRepo:       userRepo,
		tokenRepo:      tokenRepo,
		tokenGenerator: tokenGenerator,
		mailProvider:   mailProvider,
	}
}

func (uc *forgotPasswordUseCase) Execute(ctx context.Context, req dto.ForgotPasswordRequest) error {
	emailVO, err := valueobject.NewEmail(req.Email)
	if err != nil {
		return domain.ErrInvalidEmail.Wrap(err)
	}

	email := emailVO.String()

	user, err := uc.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil
		}
		return domain.ErrInternalServer.Wrap(err)
	}

	allowed, err := uc.tokenRepo.TryAcquireResendCooldown(ctx, user.ID, forgotPasswordCooldownTTL)
	if err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	if !allowed {
		return domain.ErrRateLimitExceeded.Wrap(errors.New("forgot-password resend cooldown not elapsed"))
	}

	rawToken := uc.tokenGenerator.Generate(user.ID)
	hashedToken := uc.tokenGenerator.Hash(rawToken)

	if err := uc.tokenRepo.Save(ctx, hashedToken, user.ID, forgotPasswordTokenTTL); err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	if err := uc.mailProvider.SendForgotPasswordEmail(ctx, user.Email, rawToken); err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	return nil
}
