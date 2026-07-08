package domain

import "go-judge-system/pkg/response"

var (
	ErrPermissionDenied            = response.NewAppError(response.CodeForbidden, "Permission denied", nil)
	ErrProblemNotFound             = response.NewAppError(response.CodeNotFound, "Problem not found", nil)
	ErrProblemAlreadyExists        = response.NewAppError(response.CodeConflict, "Problem already exists", nil)
	ErrTagNotFound                 = response.NewAppError(response.CodeNotFound, "Tag not found", nil)
	ErrTagAlreadyExists            = response.NewAppError(response.CodeConflict, "Tag already exists", nil)
	ErrTagUsedByPublishedProblem   = response.NewAppError(response.CodeConflict, "Tag is used by published problems", nil)
	ErrInvalidDifficulty           = response.NewAppError(response.CodeBadRequest, "Invalid difficulty", nil)
	ErrProblemContainsInactiveTags = response.NewAppError(response.CodeBadRequest, "problem contains inactive tags", nil)
	ErrForbidden                   = response.NewAppError(response.CodeForbidden, "You are not allowed to perform this action", nil)
	ErrInvalidTestCase             = response.NewAppError(response.CodeBadRequest, "Invalid test case", nil)
	ErrTestCaseNotFound            = response.NewAppError(response.CodeNotFound, "Test case not found", nil)
	ErrTestCaseAlreadyExists       = response.NewAppError(response.CodeConflict, "Test case already exists", nil)
	ErrInvalidTestCaseOrder        = response.NewAppError(response.CodeBadRequest, "Invalid test case order", nil)
	ErrInvalidTimeLimit            = response.NewAppError(response.CodeBadRequest, "Invalid time limit", nil)
	ErrInvalidMemoryLimit          = response.NewAppError(response.CodeBadRequest, "Invalid memory limit", nil)
	ErrInvalidProblemSlug          = response.NewAppError(response.CodeBadRequest, "Invalid problem slug", nil)
	ErrInvalidInput                = response.NewAppError(response.CodeBadRequest, "Invalid input", nil)
	ErrNotOwner                    = response.NewAppError(response.CodeForbidden, "You are not the owner of this problem", nil)
	ErrInternalServer              = response.NewAppError(response.CodeInternalServer, "Internal Server Error", nil)
)
