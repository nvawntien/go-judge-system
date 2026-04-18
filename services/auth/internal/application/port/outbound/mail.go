package outbound

import "context"

type MailProvider interface {
	SendVerificationEmail(ctx context.Context, email, token string) error
	SendPasswordResetEmail(ctx context.Context, email, token string) error
}
