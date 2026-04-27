package auth

import (
	"context"
	"errors"
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
}

func NewResetPasswordUseCase(
	userRepo outbound.UserRepository,
	tokenRepo outbound.TokenRepository,
	tokenGenerator outbound.TokenGenerator,
	passwordEncoder outbound.PasswordEncoder,
) inbound.ResetPasswordUseCase {
	return &resetPasswordUseCase{
		userRepo:        userRepo,
		tokenRepo:       tokenRepo,
		tokenGenerator:  tokenGenerator,
		passwordEncoder: passwordEncoder,
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
	// Find the associated user ID
	userID, err := uc.tokenRepo.FindByToken(ctx, hashedToken)
	if err != nil {
		return domain.ErrInvalidOrExpiredToken
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

	user.UpdatePassword(passwordVO)
	// Update the user's password
	if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	// Clean up token — non-critical
	_ = uc.tokenRepo.Delete(ctx, hashedToken)

	return nil
}
