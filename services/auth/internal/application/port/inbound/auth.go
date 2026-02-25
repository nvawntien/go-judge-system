package inbound

import (
	"context"
	"go-judge-system/services/auth/internal/application/dto"
)

type RegisterUseCase interface {
	Execute(ctx context.Context, req dto.RegisterRequest) error
}

type VerifyOTPUseCase interface {
	Execute(ctx context.Context, req dto.VerifyOTPRequest) error
}

type ResendOTPUseCase interface {
	Execute(ctx context.Context, req dto.ResendOTPRequest) error
}

type OTPUseCase interface {
	RequestOTP(ctx context.Context, purpose, identifier string) error
	VerifyOTP(ctx context.Context, purpose, identifier string, otp string) error
	Cleanup(ctx context.Context, purpose, identifier string)
}