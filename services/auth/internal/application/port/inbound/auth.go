package inbound

import (
	"context"
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
