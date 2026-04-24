package auth

import (
	"context"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
)

type refreshTokenUseCase struct {
	jwt outbound.JWTProvider
}

func NewRefreshTokenUseCase(jwt outbound.JWTProvider) inbound.RefreshTokenUseCase {
	return &refreshTokenUseCase{jwt: jwt}
}

func (uc *refreshTokenUseCase) Execute(ctx context.Context, refreshToken string) (*dto.LoginResponse, error) {
	id, username, role, err := uc.jwt.VerifyRefreshToken(ctx, refreshToken)
	if err != nil {
		return &dto.LoginResponse{}, domain.ErrInvalidOrExpiredToken.Wrap(err)
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
		AccessToken:  accessToken,
		AccessExpire: accessExpire,
		RefreshToken: newRefreshToken,
		RefreshExpire: refreshExpire,
	}, nil
}
