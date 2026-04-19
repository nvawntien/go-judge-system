package auth

import (
	"context"
	"errors"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"
	"strings"

	"go.uber.org/zap"
)

type loginUseCase struct {
	userRepo        outbound.UserRepository
	passwordEncoder outbound.PasswordEncoder
	jwtProvider     outbound.JWTProvider
	logger          *zap.Logger
}

func NewLoginUseCase(userRepo outbound.UserRepository, passwordEncoder outbound.PasswordEncoder, jwtProvider outbound.JWTProvider, logger *zap.Logger) inbound.LoginUseCase {
	return &loginUseCase{userRepo: userRepo, passwordEncoder: passwordEncoder, jwtProvider: jwtProvider, logger: logger}
}

func (uc *loginUseCase) Execute(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := uc.resolveUser(ctx, req.Identifier)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			uc.logger.Warn("login failed: user not found", zap.String("identifier", req.Identifier))
			return nil, domain.ErrInvalidCredentials
		}
		uc.logger.Error("failed to resolve user by identifier", zap.String("identifier", req.Identifier), zap.Error(err))
		return nil, domain.ErrInternalServer.Wrap(err)
	}

	if !user.IsActive {
		uc.logger.Warn("login failed: user not active", zap.String("identifier", req.Identifier))
		return nil, domain.ErrUserInactive
	}

	if check := uc.passwordEncoder.ComparePasswords(user.Password, []byte(req.Password)); !check {
		uc.logger.Warn("login failed: invalid password", zap.String("identifier", req.Identifier))
		return nil, domain.ErrInvalidCredentials
	}

	accessToken, accessExpire, err := uc.jwtProvider.GenerateAccessToken(ctx, user.ID, user.Username, user.Role)
	if err != nil {
		uc.logger.Error("failed to generate access token", 
			zap.String("user_id", user.ID), 
			zap.String("username", user.Username),
			zap.String("role", user.Role),
			zap.Error(err),
		)
		return nil, domain.ErrInternalServer.Wrap(err)
	}
			
	refreshToken, refreshExpire, err := uc.jwtProvider.GenerateRefreshToken(ctx, user.ID, user.Username, user.Role)
	if err != nil {
		uc.logger.Error("failed to generate refresh token", 
			zap.String("user_id", user.ID), 
			zap.String("username", user.Username),
			zap.String("role", user.Role),
			zap.Error(err),
		)
		return nil, domain.ErrInternalServer.Wrap(err)
	}

	uc.logger.Info("user logged in successfully", zap.String("user_id", user.ID), zap.String("username", user.Username))
	
	return &dto.LoginResponse{
		AccessToken:  accessToken,
		AccessExpire: accessExpire,
		RefreshToken: refreshToken,
		RefreshExpire: refreshExpire,
	}, nil

}

func (uc *loginUseCase) resolveUser(ctx context.Context, identifier string) (*entity.User, error) {
	if strings.Contains(identifier, "@") {
		return uc.userRepo.GetUserByEmail(ctx, identifier)
	}
	return uc.userRepo.GetUserByUsername(ctx, identifier)
}
