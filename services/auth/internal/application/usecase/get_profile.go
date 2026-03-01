package usecase

import (
	"context"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"

	"go.uber.org/zap"
)

type getProfileUseCase struct {
	userRepo outbound.UserRepository
	logger   *zap.Logger
}

func NewGetProfileUseCase(userRepo outbound.UserRepository, logger *zap.Logger) inbound.GetProfileUseCase {
	return &getProfileUseCase{userRepo: userRepo, logger: logger}
}

func (uc *getProfileUseCase) Execute(ctx context.Context, username string) (dto.ProfileResponse, error) {
	user, err := uc.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		uc.logger.Error("failed to get user by username", zap.Error(err))
		return dto.ProfileResponse{}, err
	}

	return dto.ProfileResponse{
		Username:  user.Username,
		Email:     user.Email.String(),
		Rating:    user.Rating,
		CreatedAt: user.CreatedAt,
	}, nil
}
