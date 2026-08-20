package auth

import (
	"context"
	"errors"
	"go-judge-system/pkg/config"
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
	abuse          authAbuse
}

func NewResendVerificationUseCaseWithAbuse(userRepo outbound.UserRepository, mailProvider outbound.MailProvider, tokenGenerator outbound.TokenGenerator, tokenRepo outbound.TokenRepository, limiter outbound.AuthAbuseLimiter, policy config.AuthAbuseConfig) inbound.ResendVerificationUseCase {
	return &resendVerificationUseCase{userRepo: userRepo, mailProvider: mailProvider, tokenGenerator: tokenGenerator, tokenRepo: tokenRepo, abuse: authAbuse{limiter: limiter, policy: policy}}
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
		return domain.ErrInvalidEmail
	}

	email := emailVO.String()
	if uc.abuse.limiter != nil {
		// Always return the generic success response to preserve account privacy.
		if _, err := uc.abuse.allow(ctx, "verify-email-send:ip:hour", req.ClientIP, uc.abuse.policy.MailIPHourlyLimit, time.Hour); err != nil {
			return nil
		}
		if _, err := uc.abuse.allow(ctx, "verify-email-send:ip:day", req.ClientIP, uc.abuse.policy.MailIPDailyLimit, 24*time.Hour); err != nil {
			return nil
		}
	}

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
	if uc.abuse.limiter != nil {
		if _, err := uc.abuse.allow(ctx, "verify-email-send:account:cooldown", user.ID, 1, uc.abuse.policy.EmailCooldown); err != nil {
			return nil
		}
		if _, err := uc.abuse.allow(ctx, "verify-email-send:account:hour", user.ID, uc.abuse.policy.ResendAccountHourlyLimit, time.Hour); err != nil {
			return nil
		}
		if _, err := uc.abuse.allow(ctx, "verify-email-send:account:day", user.ID, uc.abuse.policy.ResendAccountDailyLimit, 24*time.Hour); err != nil {
			return nil
		}
	} else {

		allowed, err := uc.tokenRepo.TryAcquireResendCooldown(ctx, outbound.TokenPurposeVerifyEmail, user.ID, resendVerificationCooldownTTL)
		if err != nil {
			return domain.ErrInternalServer.Wrap(err)
		}
		if !allowed {
			return nil
		}
	}

	rawToken := uc.tokenGenerator.Generate(user.ID)
	hashedToken := uc.tokenGenerator.Hash(rawToken)

	if err := uc.tokenRepo.Save(ctx, outbound.TokenPurposeVerifyEmail, hashedToken, user.ID, verificationTokenTTL); err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	// Send verification email — failure is non-critical
	_ = uc.mailProvider.SendVerificationEmail(ctx, user.Email, rawToken)

	return nil
}
