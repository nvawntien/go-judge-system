package handler

import "go-judge-system/services/submission/internal/adapter/inbound/http/handler/submission"

type SubmissionHandler struct {
	CreateSubmission       *submission.CreateSubmissionHandler
	ListMySubmissions      *submission.ListMySubmissionsHandler
	ListProblemSubmissions *submission.ListProblemSubmissionsHandler
	GetMySubmission        *submission.GetMySubmissionHandler
}

func NewSubmissionHandler(
	createSubmission *submission.CreateSubmissionHandler,
	listMySubmissions *submission.ListMySubmissionsHandler,
	listProblemSubmissions *submission.ListProblemSubmissionsHandler,
	getMySubmission *submission.GetMySubmissionHandler,
) *SubmissionHandler {
	return &SubmissionHandler{
		CreateSubmission:       createSubmission,
		ListMySubmissions:      listMySubmissions,
		ListProblemSubmissions: listProblemSubmissions,
		GetMySubmission:        getMySubmission,
	}
}
