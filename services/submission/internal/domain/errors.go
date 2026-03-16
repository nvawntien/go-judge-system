package domain

import "go-judge-system/pkg/response"

var (
	ErrSubmissionNotFound = response.NewAppError(response.CodeNotFound, "Submission not found", nil)
	ErrInvalidStatus      = response.NewAppError(response.CodeBadRequest, "Invalid status", nil)
	ErrInvalidLanguage    = response.NewAppError(response.CodeBadRequest, "Invalid language", nil)
	ErrInvalidSourceCode  = response.NewAppError(response.CodeBadRequest, "Invalid source code", nil)
	ErrForbidden          = response.NewAppError(response.CodeForbidden, "You are not allowed to perform this action", nil)
	ErrInternalServer     = response.NewAppError(response.CodeInternalServer, "Internal Server Error", nil)
)
