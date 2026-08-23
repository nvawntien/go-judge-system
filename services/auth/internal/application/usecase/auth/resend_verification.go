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

func NewResendVerificationUseCaseWithAbuse(userRepo outbound.UserRepository, mailProvider outbound.MailProvider, tokenGenerator outbound.TokenGenerator, tokenRepo outbound.TokenRepository, admission outbound.AbuseAdmission, policy config.AuthAbuseConfig) inbound.ResendVerificationUseCase {
	return &resendVerificationUseCase{userRepo: userRepo, mailProvider: mailProvider, tokenGenerator: tokenGenerator, tokenRepo: tokenRepo, abuse: authAbuse{admission: admission, policy: policy}}
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
	var clientIP string
	if uc.abuse.admission != nil {
		clientIP, err = uc.abuse.clientIP(ctx)
		if err != nil {
			return nil
		}
	}

	user, err := uc.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		// Do not leak whether an email exists in the system.
		if errors.Is(err, domain.ErrUserNotFound) {
			uc.acquireGenericRequest(ctx, clientIP, email)
			return nil
		}

		return domain.ErrInternalServer.Wrap(err)
	}

	// User is already active: keep response generic to avoid account enumeration.
	if user.IsActive {
		uc.acquireGenericRequest(ctx, clientIP, email)
		return nil
	}
	if uc.abuse.admission != nil {
		result, err := uc.abuse.acquire(ctx, outbound.AdmissionRequest{
			Scopes: append(emailRequestScopes("resend", clientIP, email, uc.abuse.policy.ResendIPAccountLimit, uc.abuse.policy.ResendBroadIPHourlyLimit, uc.abuse.policy.ResendBroadIPDailyLimit, uc.abuse.policy.MailHourlyWindow, uc.abuse.policy.MailDailyWindow),
				outbound.AdmissionScope{Purpose: "resend:account-hour", Scope: user.ID, Limit: uc.abuse.policy.ResendAccountHourlyLimit, Window: uc.abuse.policy.MailHourlyWindow},
				outbound.AdmissionScope{Purpose: "resend:account-day", Scope: user.ID, Limit: uc.abuse.policy.ResendAccountDailyLimit, Window: uc.abuse.policy.MailDailyWindow},
			),
			Cooldown: &outbound.CooldownScope{Purpose: "resend:cooldown", Scope: user.ID, Duration: uc.abuse.policy.EmailCooldown},
		})
		if err != nil || !result.Allowed {
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

func (uc *resendVerificationUseCase) acquireGenericRequest(ctx context.Context, clientIP, email string) {
	if uc.abuse.admission == nil {
		return
	}
	_, _ = uc.abuse.acquire(ctx, outbound.AdmissionRequest{Scopes: emailRequestScopes("resend", clientIP, email, uc.abuse.policy.ResendIPAccountLimit, uc.abuse.policy.ResendBroadIPHourlyLimit, uc.abuse.policy.ResendBroadIPDailyLimit, uc.abuse.policy.MailHourlyWindow, uc.abuse.policy.MailDailyWindow)})
}
