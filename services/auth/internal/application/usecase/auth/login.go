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
)

type loginUseCase struct {
	userRepo        outbound.UserRepository
	passwordEncoder outbound.PasswordEncoder
	jwtProvider     outbound.JWTProvider
}

func NewLoginUseCase(userRepo outbound.UserRepository, passwordEncoder outbound.PasswordEncoder, jwtProvider outbound.JWTProvider) inbound.LoginUseCase {
	return &loginUseCase{userRepo: userRepo, passwordEncoder: passwordEncoder, jwtProvider: jwtProvider}
}

func (uc *loginUseCase) Execute(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := uc.resolveUser(ctx, req.Identifier)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, domain.ErrInternalServer.Wrap(err)
	}

	if !user.IsActive {
		return nil, domain.ErrUserInactive
	}

	if check := uc.passwordEncoder.ComparePasswords(user.Password, []byte(req.Password)); !check {
		return nil, domain.ErrInvalidCredentials
	}

	accessToken, accessExpire, err := uc.jwtProvider.GenerateAccessToken(ctx, user.ID, user.Username, user.Role)
	if err != nil {
		return nil, domain.ErrInternalServer.Wrap(err)
	}
			
	refreshToken, refreshExpire, err := uc.jwtProvider.GenerateRefreshToken(ctx, user.ID, user.Username, user.Role)
	if err != nil {
		return nil, domain.ErrInternalServer.Wrap(err)
	}

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
