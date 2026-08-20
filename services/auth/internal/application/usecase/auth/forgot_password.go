package auth

import (
	"context"
	"errors"
	"go-judge-system/pkg/config"
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
	abuse          authAbuse
}

func NewForgotPasswordUseCaseWithAbuse(userRepo outbound.UserRepository, tokenRepo outbound.TokenRepository, tokenGenerator outbound.TokenGenerator, mailProvider outbound.MailProvider, limiter outbound.AuthAbuseLimiter, policy config.AuthAbuseConfig) inbound.ForgotPasswordUseCase {
	return &forgotPasswordUseCase{userRepo: userRepo, tokenRepo: tokenRepo, tokenGenerator: tokenGenerator, mailProvider: mailProvider, abuse: authAbuse{limiter: limiter, policy: policy}}
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
		return domain.ErrInvalidEmail
	}

	email := emailVO.String()
	if uc.abuse.limiter != nil {
		if _, err := uc.abuse.allow(ctx, "forgot-password:ip:hour", req.ClientIP, uc.abuse.policy.MailIPHourlyLimit, time.Hour); err != nil {
			return nil
		}
		if _, err := uc.abuse.allow(ctx, "forgot-password:ip:day", req.ClientIP, uc.abuse.policy.MailIPDailyLimit, 24*time.Hour); err != nil {
			return nil
		}
	}

	user, err := uc.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil
		}
		return domain.ErrInternalServer.Wrap(err)
	}
	if uc.abuse.limiter != nil {
		if _, err := uc.abuse.allow(ctx, "forgot-password:account:cooldown", user.ID, 1, uc.abuse.policy.EmailCooldown); err != nil {
			return nil
		}
		if _, err := uc.abuse.allow(ctx, "forgot-password:account:hour", user.ID, uc.abuse.policy.ForgotAccountHourlyLimit, time.Hour); err != nil {
			return nil
		}
		if _, err := uc.abuse.allow(ctx, "forgot-password:account:day", user.ID, uc.abuse.policy.ForgotAccountDailyLimit, 24*time.Hour); err != nil {
			return nil
		}
	} else {

		allowed, err := uc.tokenRepo.TryAcquireResendCooldown(ctx, outbound.TokenPurposeResetPassword, user.ID, forgotPasswordCooldownTTL)
		if err != nil {
			return domain.ErrInternalServer.Wrap(err)
		}
		if !allowed {
			return nil
		}
	}

	rawToken := uc.tokenGenerator.Generate(user.ID)
	hashedToken := uc.tokenGenerator.Hash(rawToken)

	if err := uc.tokenRepo.Save(ctx, outbound.TokenPurposeResetPassword, hashedToken, user.ID, forgotPasswordTokenTTL); err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	if err := uc.mailProvider.SendForgotPasswordEmail(ctx, user.Email, rawToken); err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	return nil
}
