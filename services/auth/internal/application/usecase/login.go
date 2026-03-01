package usecase

import (
	"context"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"

	"go.uber.org/zap"
)

type loginUseCase struct {
	userRepo       outbound.UserRepository
	passwordHasher outbound.PasswordHasher
	jwtProvider    outbound.JWTProvider
	logger         *zap.Logger
}

func NewLoginUseCase(userRepo outbound.UserRepository, passwordHasher outbound.PasswordHasher, jwtProvider outbound.JWTProvider, logger *zap.Logger) inbound.LoginUseCase {
	return &loginUseCase{
		userRepo:       userRepo,
		passwordHasher: passwordHasher,
		jwtProvider:    jwtProvider,
		logger:         logger,
	}
}

func (uc *loginUseCase) Execute(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error) {
	user, err := uc.userRepo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return dto.LoginResponse{}, domain.ErrInvalidCredentials
		}
		uc.logger.Error("Failed to get user by username", zap.Error(err))
		return dto.LoginResponse{}, err
	}

	if !user.IsActive {
		return dto.LoginResponse{}, domain.ErrUserInactive
	}

	compare, err := uc.passwordHasher.Compare(user.Password, req.Password)
	if err != nil {
		uc.logger.Error("Failed to compare password", zap.Error(err))
		return dto.LoginResponse{}, err
	}
	if !compare {
		return dto.LoginResponse{}, domain.ErrInvalidCredentials
	}

	accessToken, accessExpire, err := uc.jwtProvider.GenerateAccessToken(ctx, user.ID, user.Username, user.Role)
	if err != nil {
		uc.logger.Error("Failed to generate access token", zap.Error(err))
		return dto.LoginResponse{}, err
	}

	refreshToken, refreshExpire, err := uc.jwtProvider.GenerateRefreshToken(ctx, user.ID, user.Username, user.Role)
	if err != nil {
		uc.logger.Error("Failed to generate refresh token", zap.Error(err))
		return dto.LoginResponse{}, err
	}

	return dto.LoginResponse{
		AccessToken:   accessToken,
		AccessExpire:  accessExpire,
		RefreshToken:  refreshToken,
		RefreshExpire: refreshExpire,
	}, nil
}
