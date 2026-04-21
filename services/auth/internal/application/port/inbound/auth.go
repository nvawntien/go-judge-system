package inbound

import (
	"context"
	"go-judge-system/pkg/auth"
	"go-judge-system/services/auth/internal/application/dto"
)

type RegisterUseCase interface {
	Execute(ctx context.Context, req dto.RegisterRequest) error
}

type VerifyEmailUseCase interface {
	Execute(ctx context.Context, req dto.VerifyEmailRequest) error
}

type ResendVerificationUseCase interface {
	Execute(ctx context.Context, req dto.ResendVerificationRequest) error
}

type LoginUseCase interface {
	Execute(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error)
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
