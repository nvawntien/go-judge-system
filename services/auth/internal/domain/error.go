package domain

import "errors"

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidEmail      = errors.New("invalid email format")
	ErrPasswordTooShort  = errors.New("password must be at least 8 characters")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInternalServer    = errors.New("internal server error")
	ErrUserBlocked       = errors.New("you are temporarily blocked due to multiple OTP requests")
	ErrRateLimitExceeded = errors.New("you have exceeded the maximum requests per second")
	ErrOTPNotFound       = errors.New("OTP expired or not found")
	ErrOTPInvalid        = errors.New("invalid OTP")
)
