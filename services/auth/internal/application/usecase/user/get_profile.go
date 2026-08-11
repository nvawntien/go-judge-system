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
	if !user.IsActive || user.IsSuspended {
		return dto.GetProfileResponse{}, domain.ErrUserNotFound
	}

	return toGetProfileResponse(user), nil
}
