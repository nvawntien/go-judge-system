package user

import (
	"context"
	"errors"

	pkgAuth "go-judge-system/pkg/auth"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
)

type getMeUseCase struct {
	userRepo outbound.UserRepository
}

func NewGetMeUseCase(userRepo outbound.UserRepository) inbound.GetMeUseCase {
	return &getMeUseCase{userRepo: userRepo}
}

func (uc *getMeUseCase) Execute(ctx context.Context, claims pkgAuth.Claims) (*dto.GetMeResponse, error) {
	user, err := uc.userRepo.GetUserById(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, domain.ErrInternalServer.Wrap(err)
	}

	return &dto.GetMeResponse{
		ID:        user.ID,
		FullName:  user.FullName,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		Rating:    user.Rating,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}
