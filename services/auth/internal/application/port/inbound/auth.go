package inbound

import (
	"context"
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
	Execute(ctx context.Context, id string, req dto.ChangePasswordRequest) error
}

type RefreshTokenUseCase interface {
	Execute(ctx context.Context, refreshToken string) (dto.LoginResponse, error)
}

type GetProfileUseCase interface {
	Execute(ctx context.Context, username string) (dto.ProfileResponse, error)
}

type OTPUseCase interface {
	RequestOTP(ctx context.Context, purpose, identifier string) error
	VerifyOTP(ctx context.Context, purpose, identifier string, otp string) error
	Cleanup(ctx context.Context, purpose, identifier string)
}
