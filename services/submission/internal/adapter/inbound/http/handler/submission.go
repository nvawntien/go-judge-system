package handler

import "go-judge-system/services/submission/internal/adapter/inbound/http/handler/submission"

type SubmissionHandler struct {
	CreateSubmission *submission.CreateSubmissionHandler
	ListSubmissions  *submission.ListSubmissionsHandler
	GetMySubmission  *submission.GetMySubmissionHandler
}

func NewSubmissionHandler(
	createSubmission *submission.CreateSubmissionHandler,
	listSubmissions *submission.ListSubmissionsHandler,
	getMySubmission *submission.GetMySubmissionHandler,
) *SubmissionHandler {
	return &SubmissionHandler{
		CreateSubmission: createSubmission,
		ListSubmissions:  listSubmissions,
		GetMySubmission:  getMySubmission,
	}
}
