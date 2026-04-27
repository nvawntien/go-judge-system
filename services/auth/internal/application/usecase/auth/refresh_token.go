package auth

import (
	"context"
	pkgauth "go-judge-system/pkg/auth"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
)

type refreshTokenUseCase struct {
	jwt            outbound.JWTProvider
	logoutAllStore pkgauth.LogoutAllIATStore
}

func NewRefreshTokenUseCase(jwt outbound.JWTProvider, logoutAllStore pkgauth.LogoutAllIATStore) inbound.RefreshTokenUseCase {
	return &refreshTokenUseCase{
		jwt:            jwt,
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
