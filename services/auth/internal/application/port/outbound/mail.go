package outbound

import "context"

type MailProvider interface {
	SendOTP(ctx context.Context, email string, otp string) error
}
