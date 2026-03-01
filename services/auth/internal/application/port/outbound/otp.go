package outbound

import "context"

type OTPService interface {
	RequestOTP(ctx context.Context, purpose, identifier string) error
	VerifyOTP(ctx context.Context, purpose, identifier string, otp string) error
	Cleanup(ctx context.Context, purpose, identifier string)
}
