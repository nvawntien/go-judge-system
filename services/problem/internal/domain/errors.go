package domain

import "go-judge-system/pkg/response"

var (
	ErrProblemNotFound       = response.NewAppError(response.CodeNotFound, "Problem not found", nil)
	ErrProblemAlreadyExists  = response.NewAppError(response.CodeConflict, "Problem already exists", nil)
	ErrInvalidDifficulty     = response.NewAppError(response.CodeBadRequest, "Invalid difficulty", nil)
	ErrForbidden             = response.NewAppError(response.CodeForbidden, "You are not allowed to perform this action", nil)
	ErrInvalidTestCase       = response.NewAppError(response.CodeBadRequest, "Invalid test case", nil)
	ErrTestCaseNotFound      = response.NewAppError(response.CodeNotFound, "Test case not found", nil)
	ErrTestCaseAlreadyExists = response.NewAppError(response.CodeConflict, "Test case already exists", nil)
	ErrInvalidTestCaseOrder  = response.NewAppError(response.CodeBadRequest, "Invalid test case order", nil)
	ErrInvalidTimeLimit      = response.NewAppError(response.CodeBadRequest, "Invalid time limit", nil)
	ErrInvalidMemoryLimit    = response.NewAppError(response.CodeBadRequest, "Invalid memory limit", nil)
	ErrInvalidProblemID      = response.NewAppError(response.CodeBadRequest, "Invalid problem ID", nil)
	ErrInvalidInput          = response.NewAppError(response.CodeBadRequest, "Invalid input", nil)
	ErrNotOwner              = response.NewAppError(response.CodeForbidden, "You are not the owner of this problem", nil)
	ErrInternalServer        = response.NewAppError(response.CodeInternalServer, "Internal Server Error", nil)
)
