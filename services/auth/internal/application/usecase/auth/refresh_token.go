package auth

import (
	"context"
	"errors"
	pkgauth "go-judge-system/pkg/auth"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
)

type refreshTokenUseCase struct {
	jwt            outbound.JWTProvider
	userRepo       outbound.UserRepository
	logoutAllStore pkgauth.LogoutAllIATStore
}

func NewRefreshTokenUseCase(jwt outbound.JWTProvider, userRepo outbound.UserRepository, logoutAllStore pkgauth.LogoutAllIATStore) inbound.RefreshTokenUseCase {
	return &refreshTokenUseCase{
		jwt:            jwt,
		userRepo:       userRepo,
		logoutAllStore: logoutAllStore,
	}
}

func (uc *refreshTokenUseCase) Execute(ctx context.Context, refreshToken string) (*dto.LoginResponse, error) {
	id, username, role, refreshTokenIAT, err := uc.jwt.VerifyRefreshToken(ctx, refreshToken)
	if err != nil {
		return &dto.LoginResponse{}, domain.ErrInvalidOrExpiredToken
	}

	logoutAllIAT, err := uc.logoutAllStore.GetLogoutAllIAT(ctx, id)
	if err != nil {
		return &dto.LoginResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	if logoutAllIAT > 0 && refreshTokenIAT <= logoutAllIAT {
		return &dto.LoginResponse{}, domain.ErrInvalidOrExpiredToken
	}

	user, err := uc.userRepo.GetUserById(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return &dto.LoginResponse{}, domain.ErrInvalidOrExpiredToken
		}
		return &dto.LoginResponse{}, domain.ErrInternalServer.Wrap(err)
	}
	if !user.IsActive {
		return &dto.LoginResponse{}, domain.ErrUserInactive
	}
	if user.IsSuspended {
		return &dto.LoginResponse{}, domain.ErrUserSuspended
	}

	accessToken, accessExpire, err := uc.jwt.GenerateAccessToken(ctx, id, username, role)
	if err != nil {
		return &dto.LoginResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	newRefreshToken, refreshExpire, err := uc.jwt.GenerateRefreshToken(ctx, id, username, role)
	if err != nil {
		return &dto.LoginResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	return &dto.LoginResponse{
		AccessToken:   accessToken,
		AccessExpire:  accessExpire,
		RefreshToken:  newRefreshToken,
		RefreshExpire: refreshExpire,
	}, nil
}
