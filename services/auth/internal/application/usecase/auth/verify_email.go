package auth

import (
	"context"
	"errors"

	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
)

type verifyEmail struct {
	tokenGenerator outbound.TokenGenerator
	tokenRepo      outbound.TokenRepository
	userRepo       outbound.UserRepository
}

func NewVerifyEmailUseCase(
	tokenGenerator outbound.TokenGenerator,
	tokenRepo outbound.TokenRepository,
	userRepo outbound.UserRepository,
) inbound.VerifyEmailUseCase {
	return &verifyEmail{
		tokenGenerator: tokenGenerator,
		tokenRepo:      tokenRepo,
		userRepo:       userRepo,
	}
}

func (uc *verifyEmail) Execute(ctx context.Context, req dto.VerifyEmailRequest) error {
	// Hash the raw token to look up in Redis
	hashedToken := uc.tokenGenerator.Hash(req.Token)

	userID, err := uc.tokenRepo.FindByToken(ctx, outbound.TokenPurposeVerifyEmail, hashedToken)
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

	// Check if already active
	if user.IsActive {
		return domain.ErrUserAlreadyActive
	}

	// Consume immediately before activation so concurrent verification requests
	// cannot replay the same token. Reset-token state is purpose-isolated.
	if _, err := uc.tokenRepo.Consume(ctx, outbound.TokenPurposeVerifyEmail, hashedToken); err != nil {
		if errors.Is(err, domain.ErrInvalidOrExpiredToken) {
			return domain.ErrInvalidOrExpiredToken
		}
		return domain.ErrInternalServer.Wrap(err)
	}

	// Activate user
	user.Activate()
	if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	return nil
}
