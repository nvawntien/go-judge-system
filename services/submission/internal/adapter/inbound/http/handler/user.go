package handler

import "go-judge-system/services/submission/internal/adapter/inbound/http/handler/user"

type UserHandler struct {
	CreateSubmission  *user.CreateSubmissionHandler
	GetSubmission     *user.GetSubmissionHandler
	ListMySubmissions *user.ListMySubmissionsHandler
}

func NewUserHandler(
	createSubmission *user.CreateSubmissionHandler,
	getSubmission *user.GetSubmissionHandler,
	listMySubmissions *user.ListMySubmissionsHandler,
) *UserHandler {
	return &UserHandler{
		CreateSubmission:  createSubmission,
		GetSubmission:     getSubmission,
		ListMySubmissions: listMySubmissions,
	}
}
