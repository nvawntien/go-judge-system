package auth

import (
	"context"
	"errors"
	pkgauth "go-judge-system/pkg/auth"
	"go-judge-system/pkg/config"
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
	logoutAllStore  pkgauth.LogoutAllIATStore
	abuse           authAbuse
}

func NewLoginUseCase(userRepo outbound.UserRepository, passwordEncoder outbound.PasswordEncoder, jwtProvider outbound.JWTProvider, logoutAllStore pkgauth.LogoutAllIATStore) inbound.LoginUseCase {
	return &loginUseCase{userRepo: userRepo, passwordEncoder: passwordEncoder, jwtProvider: jwtProvider, logoutAllStore: logoutAllStore}
}

func NewLoginUseCaseWithAbuse(userRepo outbound.UserRepository, passwordEncoder outbound.PasswordEncoder, jwtProvider outbound.JWTProvider, logoutAllStore pkgauth.LogoutAllIATStore, limiter outbound.AuthAbuseLimiter, policy config.AuthAbuseConfig) inbound.LoginUseCase {
	return &loginUseCase{userRepo: userRepo, passwordEncoder: passwordEncoder, jwtProvider: jwtProvider, logoutAllStore: logoutAllStore, abuse: authAbuse{limiter: limiter, policy: policy}}
}

func (uc *loginUseCase) Execute(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	identifier := normalizedIdentifier(req.Identifier)
	if uc.abuse.limiter != nil {
		clientIP, err := uc.abuse.clientIP(ctx)
		if err != nil {
			return nil, err
		}
		if _, err := uc.abuse.allow(ctx, "login:ip", clientIP, uc.abuse.policy.LoginIPLimit, uc.abuse.policy.LoginWindow); err != nil {
			return nil, err
		}
		if _, err := uc.abuse.allow(ctx, "login:identifier", identifier, uc.abuse.policy.LoginIdentifierLimit, uc.abuse.policy.LoginWindow); err != nil {
			return nil, err
		}
	}
	user, err := uc.resolveUser(ctx, identifier)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, domain.ErrInternalServer.Wrap(err)
	}

	if !user.IsActive {
		return nil, domain.ErrUserInactive
	}
	if user.IsSuspended {
		return nil, domain.ErrUserSuspended
	}

	if check := uc.passwordEncoder.ComparePasswords(user.Password, []byte(req.Password)); !check {
		return nil, domain.ErrInvalidCredentials
	}
	if uc.abuse.limiter != nil {
		_ = uc.abuse.limiter.Reset(ctx, "login:identifier", identifier)
	}

	waited, err := waitForTokenIssuedAfterInvalidation(ctx, uc.logoutAllStore, user.ID)
	if err != nil {
		return nil, domain.ErrInternalServer.Wrap(err)
	}
	if waited {
		user, err = uc.userRepo.GetUserById(ctx, user.ID)
		if err != nil {
			return nil, domain.ErrInternalServer.Wrap(err)
		}
		if !user.IsActive {
			return nil, domain.ErrUserInactive
		}
		if user.IsSuspended {
			return nil, domain.ErrUserSuspended
		}
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
		AccessToken:   accessToken,
		AccessExpire:  accessExpire,
		RefreshToken:  refreshToken,
		RefreshExpire: refreshExpire,
	}, nil

}

func (uc *loginUseCase) resolveUser(ctx context.Context, identifier string) (*entity.User, error) {
	if strings.Contains(identifier, "@") {
		return uc.userRepo.GetUserByEmail(ctx, identifier)
	}
	return uc.userRepo.GetUserByUsername(ctx, identifier)
}
