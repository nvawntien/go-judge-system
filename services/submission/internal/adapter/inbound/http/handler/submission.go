package handler

import "go-judge-system/services/submission/internal/adapter/inbound/http/handler/submission"

type SubmissionHandler struct {
	CreateSubmission *submission.CreateSubmissionHandler
}

func NewSubmissionHandler(createSubmission *submission.CreateSubmissionHandler) *SubmissionHandler {
	return &SubmissionHandler{CreateSubmission: createSubmission}
}
