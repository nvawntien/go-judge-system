package auth

import (
	"context"
	"errors"
	"time"

	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/valueobject"
)

const resendVerificationCooldownTTL = 60 * time.Second

type resendVerificationUseCase struct {
	userRepo       outbound.UserRepository
	mailProvider   outbound.MailProvider
	tokenGenerator outbound.TokenGenerator
	tokenRepo      outbound.TokenRepository
}

func NewResendVerificationUseCase(
	userRepo outbound.UserRepository,
	mailProvider outbound.MailProvider,
	tokenGenerator outbound.TokenGenerator,
	tokenRepo outbound.TokenRepository,
) inbound.ResendVerificationUseCase {
	return &resendVerificationUseCase{
		userRepo:       userRepo,
		mailProvider:   mailProvider,
		tokenGenerator: tokenGenerator,
		tokenRepo:      tokenRepo,
	}
}

func (uc *resendVerificationUseCase) Execute(ctx context.Context, req dto.ResendVerificationRequest) error {
	emailVO, err := valueobject.NewEmail(req.Email)
	if err != nil {
		return err
	}

	email := emailVO.String()
	
	user, err := uc.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		// Do not leak whether an email exists in the system.
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil
		}

		return domain.ErrInternalServer.Wrap(err)
	}

	// User is already active: keep response generic to avoid account enumeration.
	if user.IsActive {
		return nil
	}

	allowed, err := uc.tokenRepo.TryAcquireResendCooldown(ctx, user.ID, resendVerificationCooldownTTL)
	if err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	if !allowed {
		return domain.ErrRateLimitExceeded
	}

	rawToken := uc.tokenGenerator.Generate(user.ID)
	hashedToken := uc.tokenGenerator.Hash(rawToken)

	if err := uc.tokenRepo.Save(ctx, hashedToken, user.ID, verificationTokenTTL); err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	// Send verification email — failure is non-critical
	_ = uc.mailProvider.SendVerificationEmail(ctx, user.Email, rawToken)

	return nil
}
