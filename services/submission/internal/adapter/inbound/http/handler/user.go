package handler

import "go-judge-system/services/submission/internal/adapter/inbound/http/handler/user"

type UserHandler struct {
	CreateSubmission  *user.CreateSubmissionHandler
	RunCode           *user.RunCodeHandler
	GetSubmission     *user.GetSubmissionHandler
	ListMySubmissions *user.ListMySubmissionsHandler
	IssueStreamTicket *user.IssueSubmissionStreamTicketHandler
	SubmissionEvents  *user.SubmissionEventsHandler
}

func NewUserHandler(
	createSubmission *user.CreateSubmissionHandler,
	runCode *user.RunCodeHandler,
	getSubmission *user.GetSubmissionHandler,
	listMySubmissions *user.ListMySubmissionsHandler,
	issueStreamTicket *user.IssueSubmissionStreamTicketHandler,
	submissionEvents *user.SubmissionEventsHandler,
) *UserHandler {
	return &UserHandler{
		CreateSubmission:  createSubmission,
		RunCode:           runCode,
		GetSubmission:     getSubmission,
		ListMySubmissions: listMySubmissions,
		IssueStreamTicket: issueStreamTicket,
		SubmissionEvents:  submissionEvents,
	}
}
