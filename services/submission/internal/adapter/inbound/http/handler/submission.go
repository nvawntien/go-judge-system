package handler

import "go-judge-system/services/submission/internal/adapter/inbound/http/handler/submission"

type SubmissionHandler struct {
	CreateSubmission  *submission.CreateSubmissionHandler
	ListMySubmissions *submission.ListMySubmissionsHandler
	GetMySubmission   *submission.GetMySubmissionHandler
}

func NewSubmissionHandler(
	createSubmission *submission.CreateSubmissionHandler,
	listMySubmissions *submission.ListMySubmissionsHandler,
	getMySubmission *submission.GetMySubmissionHandler,
) *SubmissionHandler {
	return &SubmissionHandler{
		CreateSubmission:  createSubmission,
		ListMySubmissions: listMySubmissions,
		GetMySubmission:   getMySubmission,
	}
}
