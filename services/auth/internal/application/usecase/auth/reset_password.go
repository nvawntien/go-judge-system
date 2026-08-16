package auth

import (
	"context"
	"errors"
	"time"

	pkgauth "go-judge-system/pkg/auth"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/valueobject"
)

type resetPasswordUseCase struct {
	userRepo        outbound.UserRepository
	tokenRepo       outbound.TokenRepository
	tokenGenerator  outbound.TokenGenerator
	passwordEncoder outbound.PasswordEncoder
	logoutAllStore  pkgauth.LogoutAllIATStore
}

func NewResetPasswordUseCase(
	userRepo outbound.UserRepository,
	tokenRepo outbound.TokenRepository,
	tokenGenerator outbound.TokenGenerator,
	passwordEncoder outbound.PasswordEncoder,
	logoutAllStore pkgauth.LogoutAllIATStore,
) inbound.ResetPasswordUseCase {
	return &resetPasswordUseCase{
		userRepo:        userRepo,
		tokenRepo:       tokenRepo,
		tokenGenerator:  tokenGenerator,
		passwordEncoder: passwordEncoder,
		logoutAllStore:  logoutAllStore,
	}
}

func (uc *resetPasswordUseCase) Execute(ctx context.Context, req dto.ResetPasswordRequest) error {
	// Hash the raw token to look up in Redis
	if err := valueobject.ValidatePlainPassword(req.NewPassword); err != nil {
		return domain.ErrPasswordTooWeak
	}

	if req.NewPassword != req.ConfirmPassword {
		return domain.ErrPasswordMismatch
	}

	hashedToken := uc.tokenGenerator.Hash(req.Token)
	// Consume before side effects so a reset token cannot be replayed when a
	// downstream invalidation or database write fails.
	userID, err := uc.tokenRepo.Consume(ctx, outbound.TokenPurposeResetPassword, hashedToken)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidOrExpiredToken) {
			return domain.ErrInvalidOrExpiredToken
		}
		return domain.ErrInternalServer.Wrap(err)
	}

	// Find the user
	user, err := uc.userRepo.GetUserById(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.ErrUserNotFound
		}
		return domain.ErrInternalServer.Wrap(err)
	}

	// Encode the new password
	hashedPassword, err := uc.passwordEncoder.HashAndSalt([]byte(req.NewPassword))
	if err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	passwordVO := valueobject.NewPasswordFromHash(hashedPassword)

	// Never persist a changed password unless previously issued sessions have
	// been invalidated. A database failure leaves the cutoff in force.
	if err := uc.logoutAllStore.SetLogoutAllIAT(ctx, user.ID, time.Now().Unix()); err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	user.UpdatePassword(passwordVO)
	if err := uc.userRepo.UpdatePassword(ctx, user.ID, user.Password, user.UpdatedAt); err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	return nil
}
