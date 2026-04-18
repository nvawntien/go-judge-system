package domain

import "go-judge-system/pkg/response"

var (
	ErrDuplicateEntry        = response.NewAppError(response.CodeConflict, "duplicate entry", nil)
	ErrUserInactive          = response.NewAppError(response.CodeForbidden, "user is not active", nil)
	ErrUserAlreadyActive     = response.NewAppError(response.CodeConflict, "user is already active", nil)
	ErrEmailAlreadyExists    = response.NewAppError(response.CodeConflict, "email already exists", nil)
	ErrUsernameAlreadyExists = response.NewAppError(response.CodeConflict, "username already exists", nil)
	ErrUserNotFound          = response.NewAppError(response.CodeAccountNotFound, "user not found", nil)
	ErrInvalidEmail          = response.NewAppError(response.CodeBadRequest, "invalid email format", nil)
	ErrPasswordTooWeak       = response.NewAppError(response.CodeBadRequest, "password is too weak", nil)
	ErrUserAlreadyExists     = response.NewAppError(response.CodeConflict, "user already exists", nil)
	ErrInternalServer        = response.NewAppError(response.CodeInternalServer, "internal server error", nil)
	ErrUserBlocked           = response.NewAppError(response.CodeForbidden, "you are temporarily blocked due to multiple OTP requests", nil)
	ErrRateLimitExceeded     = response.NewAppError(response.CodeRateLimitExceeded, "you have exceeded the maximum requests per second", nil)
	ErrInvalidOrExpiredToken = response.NewAppError(response.CodeInvalidToken, "invalid or expired token", nil)
	ErrInvalidCredentials    = response.NewAppError(response.CodeUnauthorized, "invalid username or password", nil)
	ErrIncorrecOldPassword   = response.NewAppError(response.CodeInvalidPassword, "incorrect old password", nil)
	ErrForbidden             = response.NewAppError(response.CodeForbidden, "you are not allowed to perform this action", nil)
)
