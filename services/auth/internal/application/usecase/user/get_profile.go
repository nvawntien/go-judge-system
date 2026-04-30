package user

import (
	"context"
	"errors"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
)

type getProfileUseCase struct {
	userRepo outbound.UserRepository
}

func NewGetProfileUseCase(userRepo outbound.UserRepository) inbound.GetProfileUseCase {
	return &getProfileUseCase{userRepo: userRepo}
}

func (uc *getProfileUseCase) Execute(ctx context.Context, req dto.GetProfileRequest) (dto.GetProfileResponse, error) {
	user, err := uc.userRepo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return dto.GetProfileResponse{}, domain.ErrUserNotFound
		}
		return dto.GetProfileResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	return dto.GetProfileResponse{
		FullName:  user.FullName,
		Username:  user.Username,
		Email:     user.Email,
		Rating:    user.Rating,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}
