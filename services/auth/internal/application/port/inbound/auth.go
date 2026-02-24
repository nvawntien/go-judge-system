package inbound

import (
	"context"
	"go-judge-system/services/auth/internal/application/dto"
)

type RegisterUseCase interface {
	Execute(ctx context.Context, req dto.RegisterRequest) error
}

type OTPUseCase interface {
	RequestOTP(ctx context.Context, identifier string) error
	//VerifyOTP(ctx context.Context, identifier string, otp string) error
}