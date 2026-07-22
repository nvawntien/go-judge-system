package handler

import "go-judge-system/services/submission/internal/adapter/inbound/http/handler/user"

type UserHandler struct {
	CreateSubmission *user.CreateSubmissionHandler
}

func NewUserHandler(createSubmission *user.CreateSubmissionHandler) *UserHandler {
	return &UserHandler{CreateSubmission: createSubmission}
}
