package inbound

import (
	"context"
	"go-judge-system/pkg/auth"
	"go-judge-system/services/auth/internal/application/dto"
)

type RegisterUseCase interface {
	Execute(ctx context.Context, req dto.RegisterRequest) error
}

type VerifyActivationUseCase interface {
	Execute(ctx context.Context, req dto.VerifyOTPRequest) error
}

type LoginUseCase interface {
	Execute(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error)
}

type VerifyForgotPasswordUseCase interface {
	Execute(ctx context.Context, req dto.VerifyOTPRequest) (string, error)
}

type ResendOTPUseCase interface {
	Execute(ctx context.Context, req dto.ResendOTPRequest) error
}

type ForgotPasswordUseCase interface {
	Execute(ctx context.Context, req dto.ForgotPasswordRequest) error
}

type ResetPasswordUseCase interface {
	Execute(ctx context.Context, req dto.ResetPasswordRequest) error
}

type ChangePasswordUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, req dto.ChangePasswordRequest) error
}

type RefreshTokenUseCase interface {
	Execute(ctx context.Context, refreshToken string) (dto.LoginResponse, error)
}

type GetProfileUseCase interface {
	Execute(ctx context.Context, username string) (dto.ProfileResponse, error)
	ExecuteMe(ctx context.Context, claims auth.Claims) (dto.ProfileResponse, error)
	ExecutePublic(ctx context.Context, req dto.ProfileRequest) (dto.ProfileResponse, error)
}
