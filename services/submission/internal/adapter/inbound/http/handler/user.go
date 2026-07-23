package handler

import "go-judge-system/services/submission/internal/adapter/inbound/http/handler/user"

type UserHandler struct {
	CreateSubmission *user.CreateSubmissionHandler
	GetSubmission    *user.GetSubmissionHandler
}

func NewUserHandler(
	createSubmission *user.CreateSubmissionHandler,
	getSubmission *user.GetSubmissionHandler,
) *UserHandler {
	return &UserHandler{
		CreateSubmission: createSubmission,
		GetSubmission:    getSubmission,
	}
}
