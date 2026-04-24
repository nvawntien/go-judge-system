package auth

import (
	"context"
	"errors"
	"go-judge-system/pkg/auth"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/valueobject"
)

type changePasswordUseCase struct {
	userRepo        outbound.UserRepository
	passwordEncoder outbound.PasswordEncoder
}

func NewChangePasswordUseCase(
	userRepo outbound.UserRepository,
	passwordEncoder outbound.PasswordEncoder,
) inbound.ChangePasswordUseCase {
	return &changePasswordUseCase{
		userRepo:        userRepo,
		passwordEncoder: passwordEncoder,
	}
}

func (uc *changePasswordUseCase) Execute(ctx context.Context, claims auth.Claims, req dto.ChangePasswordRequest) error {
	if err := valueobject.ValidatePlainPassword(req.NewPassword); err != nil {
		return err
	}

	userID := claims.UserID

	// Find the user
	user, err := uc.userRepo.GetUserById(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.ErrUserNotFound
		}
		return domain.ErrInternalServer.Wrap(err)
	}

	// Check if the current password is correct
	if check := uc.passwordEncoder.ComparePasswords(user.Password, []byte(req.CurrentPassword)); !check {
		return domain.ErrIncorrectCurrentPassword
	}

	if req.CurrentPassword == req.NewPassword {
		return domain.ErrNewPasswordSameAsCurrent
	}

	if req.NewPassword != req.ConfirmPassword {
		return domain.ErrPasswordMismatch
	}

	// Hash the new password
	hashedPassword, err := uc.passwordEncoder.HashAndSalt([]byte(req.NewPassword))
	if err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	passwordVO := valueobject.NewPasswordFromHash(hashedPassword)

	// Update the user's password
	user.UpdatePassword(passwordVO)
	if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	return nil
}
