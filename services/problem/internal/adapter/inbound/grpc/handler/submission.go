package handler

import submissionproblem "go-judge-system/services/problem/internal/adapter/inbound/grpc/handler/submission/problem"

// SubmissionHandler groups the gRPC handlers exposed to Submission Service.
type SubmissionHandler struct {
	GetProblemForSubmission *submissionproblem.GetProblemForSubmissionHandler
}

func NewSubmissionHandler(
	getProblemForSubmission *submissionproblem.GetProblemForSubmissionHandler,
) *SubmissionHandler {
	return &SubmissionHandler{GetProblemForSubmission: getProblemForSubmission}
}
