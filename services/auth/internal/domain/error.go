package domain

import "go-judge-system/pkg/response"

var (
	ErrUserInactive          = response.NewAppError(response.CodeForbidden, "user is not active", nil)
	ErrUserAlreadyActive     = response.NewAppError(response.CodeConflict, "user is already active", nil)
	ErrUserNotFound          = response.NewAppError(response.CodeAccountNotFound, "user not found", nil)
	ErrInvalidEmail          = response.NewAppError(response.CodeParamInvalid, "invalid email format", nil)
	ErrPasswordTooShort      = response.NewAppError(response.CodeParamInvalid, "password must be at least 8 characters", nil)
	ErrUserAlreadyExists     = response.NewAppError(response.CodeConflict, "user already exists", nil)
	ErrInternalServer        = response.NewAppError(response.CodeInternalServer, "internal server error", nil)
	ErrUserBlocked           = response.NewAppError(response.CodeForbidden, "you are temporarily blocked due to multiple OTP requests", nil)
	ErrRateLimitExceeded     = response.NewAppError(response.CodeRateLimitExceeded, "you have exceeded the maximum requests per second", nil)
	ErrOTPNotFound           = response.NewAppError(response.CodeBadRequest, "OTP expired or not found", nil)
	ErrOTPInvalid            = response.NewAppError(response.CodeBadRequest, "invalid OTP", nil)
	ErrInvalidPurpose        = response.NewAppError(response.CodeBadRequest, "invalid OTP purpose", nil)
	ErrInvalidOrExpiredToken = response.NewAppError(response.CodeInvalidToken, "invalid or expired token", nil)
	ErrInvalidCredentials    = response.NewAppError(response.CodeUnauthorized, "invalid username or password", nil)
	ErrIncorrecOldPassword   = response.NewAppError(response.CodeInvalidPassword, "incorrect old password", nil)
	ErrForbidden             = response.NewAppError(response.CodeForbidden, "you are not allowed to perform this action", nil)
)
