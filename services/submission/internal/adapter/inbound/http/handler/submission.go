package handler

import "go-judge-system/services/submission/internal/adapter/inbound/http/handler/submission"

type SubmissionHandler struct {
	CreateSubmission *submission.CreateSubmissionHandler
	ListSubmissions  *submission.ListSubmissionsHandler
	GetSubmission    *submission.GetSubmissionHandler
}

func NewSubmissionHandler(
	createSubmission *submission.CreateSubmissionHandler,
	listSubmissions *submission.ListSubmissionsHandler,
	getSubmission *submission.GetSubmissionHandler,
) *SubmissionHandler {
	return &SubmissionHandler{
		CreateSubmission: createSubmission,
		ListSubmissions:  listSubmissions,
		GetSubmission:    getSubmission,
	}
}
