package usecase

import (
	"context"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"

	"go.uber.org/zap"
)

type refreshTokenUseCase struct {
	jwt    outbound.JWTProvider
	logger *zap.Logger
}

func NewRefreshTokenUseCase(jwt outbound.JWTProvider, logger *zap.Logger) inbound.RefreshTokenUseCase {
	return &refreshTokenUseCase{
		jwt:    jwt,
		logger: logger,
	}
}

func (uc *refreshTokenUseCase) Execute(ctx context.Context, refreshToken string) (dto.LoginResponse, error) {
	userID, username, role, err := uc.jwt.VerifyRefreshToken(ctx, refreshToken)
	if err != nil {
		uc.logger.Warn("invalid or expired refresh token", zap.Error(err))
		return dto.LoginResponse{}, err
	}

	accessToken, accessExpire, err := uc.jwt.GenerateAccessToken(ctx, userID, username, role)
	if err != nil {
		uc.logger.Error("failed to generate new access token", zap.Error(err))
		return dto.LoginResponse{}, err
	}

	newRefreshToken, refreshExpire, err := uc.jwt.GenerateRefreshToken(ctx, userID, username, role)
	if err != nil {
		uc.logger.Error("failed to generate new refresh token", zap.Error(err))
		return dto.LoginResponse{}, err
	}

	return dto.LoginResponse{
		AccessToken:   accessToken,
		AccessExpire:  accessExpire,
		RefreshToken:  newRefreshToken,
		RefreshExpire: refreshExpire,
	}, nil
}
