package handler

import problemhandler "go-judge-system/services/problem/internal/adapter/inbound/grpc/handler/problem"

// ProblemHandler groups actor-aware Problem query handlers.
type ProblemHandler struct {
	GetProblem *problemhandler.GetProblemHandler
}

func NewProblemHandler(getProblem *problemhandler.GetProblemHandler) *ProblemHandler {
	return &ProblemHandler{GetProblem: getProblem}
}
