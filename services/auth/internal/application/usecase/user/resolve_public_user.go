package user

import (
	"context"
	"errors"
	"strings"

	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
)

type resolvePublicUserUseCase struct {
	userRepo outbound.UserRepository
}

func NewResolvePublicUserUseCase(userRepo outbound.UserRepository) inbound.ResolvePublicUserUseCase {
	return &resolvePublicUserUseCase{userRepo: userRepo}
}

func (uc *resolvePublicUserUseCase) Execute(
	ctx context.Context,
	req dto.ResolvePublicUserRequest,
) (dto.ResolvePublicUserResponse, error) {
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return dto.ResolvePublicUserResponse{}, domain.ErrUserNotFound
	}

	user, err := uc.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return dto.ResolvePublicUserResponse{}, domain.ErrUserNotFound
		}
		return dto.ResolvePublicUserResponse{}, domain.ErrInternalServer.Wrap(err)
	}
	if !user.IsActive || user.IsSuspended {
		return dto.ResolvePublicUserResponse{}, domain.ErrUserNotFound
	}

	return dto.ResolvePublicUserResponse{UserID: user.ID, Username: user.Username}, nil
}
