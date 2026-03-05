package response

import (
	"fmt"
	"net/http"
)

// AppError is a custom error carrying a business code
type AppError struct {
	Code    int
	Message string
	Err     error // Root cause (for logging, not exposed to client)
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Is allows errors.Is() to compare by Code instead of pointer
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// Unwrap returns the underlying error for errors.As() / errors.Is() chain traversal
func (e *AppError) Unwrap() error { return e.Err }

// Wrap creates a copy of the AppError with the root cause attached
// The original variable is NOT mutated, preserving errors.Is() compatibility
func (e *AppError) Wrap(err error) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: e.Message,
		Err:     err,
	}
}

// NewAppError creates a new AppError with the given code and message
func NewAppError(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// GetHTTPStatus maps a business code to an HTTP status code
func GetHTTPStatus(code int) int {
	// Specific codes
	switch code {
	case CodeSuccess, CodeUpdated, CodeDeleted, CodeRetrieved:
		return http.StatusOK
	case CodeCreated:
		return http.StatusCreated
	case CodeParamInvalid, CodeBadRequest, CodeInvalidID, CodeInternalError:
		return http.StatusBadRequest
	case CodeUnauthorized, CodeInvalidToken, CodeTokenExpired, CodeInvalidPassword:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeAccountNotFound, CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeValidationFailed:
		return http.StatusUnprocessableEntity
	case CodeRateLimitExceeded:
		return http.StatusTooManyRequests
	case CodeInternalServer, CodeDatabaseError, CodeMongoDBError, CodeRedisError:
		return http.StatusInternalServerError
	}

	// Fallback range mapping
	switch {
	case code >= 20000 && code < 30000:
		return http.StatusOK
	case code >= 40000 && code < 41000:
		return http.StatusBadRequest
	case code >= 41000 && code < 42000:
		return http.StatusUnauthorized
	case code >= 43000 && code < 44000:
		return http.StatusForbidden
	case code >= 44000 && code < 45000:
		return http.StatusNotFound
	case code >= 49000 && code < 50000:
		return http.StatusConflict
	case code >= 50000 && code < 60000:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
