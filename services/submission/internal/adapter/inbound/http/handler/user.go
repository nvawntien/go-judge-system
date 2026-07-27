package handler

import "go-judge-system/services/submission/internal/adapter/inbound/http/handler/user"

type UserHandler struct {
	CreateSubmission  *user.CreateSubmissionHandler
	RunCode           *user.RunCodeHandler
	GetSubmission     *user.GetSubmissionHandler
	ListMySubmissions *user.ListMySubmissionsHandler
}

func NewUserHandler(
	createSubmission *user.CreateSubmissionHandler,
	runCode *user.RunCodeHandler,
	getSubmission *user.GetSubmissionHandler,
	listMySubmissions *user.ListMySubmissionsHandler,
) *UserHandler {
	return &UserHandler{
		CreateSubmission:  createSubmission,
		RunCode:           runCode,
		GetSubmission:     getSubmission,
		ListMySubmissions: listMySubmissions,
	}
}
