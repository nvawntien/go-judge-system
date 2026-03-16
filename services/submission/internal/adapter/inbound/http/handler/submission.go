package handler

import "go-judge-system/services/submission/internal/adapter/inbound/http/handler/submission"

type SubmissionHandler struct {
	CreateSubmission  *submission.CreateSubmissionHandler
	ListMySubmissions *submission.ListMySubmissionsHandler
}

func NewSubmissionHandler(
	createSubmission *submission.CreateSubmissionHandler,
	listMySubmissions *submission.ListMySubmissionsHandler,
) *SubmissionHandler {
	return &SubmissionHandler{
		CreateSubmission:  createSubmission,
		ListMySubmissions: listMySubmissions,
	}
}
