package domain

import "go-judge-system/pkg/response"

var (
	ErrSubmissionNotFound        = response.NewAppError(response.CodeNotFound, "submission not found", nil)
	ErrSubmissionForbidden       = response.NewAppError(response.CodeForbidden, "you are not allowed to access this submission", nil)
	ErrInvalidProblemID          = response.NewAppError(response.CodeBadRequest, "invalid problem ID", nil)
	ErrProblemNotFound           = response.NewAppError(response.CodeNotFound, "problem not found", nil)
	ErrProblemServiceUnavailable = response.NewAppError(response.CodeServiceUnavailable, "problem service unavailable", nil)
	ErrInvalidLanguage           = response.NewAppError(response.CodeBadRequest, "invalid submission language", nil)
	ErrInvalidSourceCode         = response.NewAppError(response.CodeBadRequest, "invalid source code", nil)
	ErrSourceCodeTooLarge        = response.NewAppError(response.CodeBadRequest, "source code is too large", nil)
	ErrInvalidSubmissionStatus   = response.NewAppError(response.CodeBadRequest, "invalid submission status", nil)
	ErrInvalidStatusTransition   = response.NewAppError(response.CodeConflict, "invalid submission status transition", nil)
	ErrInternalServer            = response.NewAppError(response.CodeInternalServer, "internal server error", nil)
)
