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

func NewForgotPasswordUseCaseWithAbuse(userRepo outbound.UserRepository, tokenRepo outbound.TokenRepository, tokenGenerator outbound.TokenGenerator, mailProvider outbound.MailProvider, admission outbound.AbuseAdmission, policy config.AuthAbuseConfig) inbound.ForgotPasswordUseCase {
	return &forgotPasswordUseCase{userRepo: userRepo, tokenRepo: tokenRepo, tokenGenerator: tokenGenerator, mailProvider: mailProvider, abuse: authAbuse{admission: admission, policy: policy}}
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
	var clientIP string
	if uc.abuse.admission != nil {
		clientIP, err = uc.abuse.clientIP(ctx)
		if err != nil {
			return nil
		}
	}

	user, err := uc.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			uc.acquireGenericRequest(ctx, clientIP, email)
			return nil
		}
		return domain.ErrInternalServer.Wrap(err)
	}
	if uc.abuse.admission != nil {
		result, err := uc.abuse.acquire(ctx, outbound.AdmissionRequest{
			Scopes: append(emailRequestScopes("forgot", clientIP, email, uc.abuse.policy.ForgotIPAccountLimit, uc.abuse.policy.ForgotBroadIPHourlyLimit, uc.abuse.policy.ForgotBroadIPDailyLimit, uc.abuse.policy.MailHourlyWindow, uc.abuse.policy.MailDailyWindow),
				outbound.AdmissionScope{Purpose: "forgot:account-hour", Scope: user.ID, Limit: uc.abuse.policy.ForgotAccountHourlyLimit, Window: uc.abuse.policy.MailHourlyWindow},
				outbound.AdmissionScope{Purpose: "forgot:account-day", Scope: user.ID, Limit: uc.abuse.policy.ForgotAccountDailyLimit, Window: uc.abuse.policy.MailDailyWindow},
			),
			Cooldown: &outbound.CooldownScope{Purpose: "forgot:cooldown", Scope: user.ID, Duration: uc.abuse.policy.EmailCooldown},
		})
		if err != nil || !result.Allowed {
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

func (uc *forgotPasswordUseCase) acquireGenericRequest(ctx context.Context, clientIP, email string) {
	if uc.abuse.admission == nil {
		return
	}
	_, _ = uc.abuse.acquire(ctx, outbound.AdmissionRequest{Scopes: emailRequestScopes("forgot", clientIP, email, uc.abuse.policy.ForgotIPAccountLimit, uc.abuse.policy.ForgotBroadIPHourlyLimit, uc.abuse.policy.ForgotBroadIPDailyLimit, uc.abuse.policy.MailHourlyWindow, uc.abuse.policy.MailDailyWindow)})
}
