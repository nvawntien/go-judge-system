package handler

import "go-judge-system/services/problem/internal/adapter/inbound/http/handler/problem"

type ProblemHandler struct {
	CreateProblem     *problem.CreateProblemHandler
	UpdateProblem     *problem.UpdateProblemHandler
	DeleteProblem     *problem.DeleteProblemHandler
	GetProblem        *problem.GetProblemHandler
	ListProblems      *problem.ListProblemsHandler
	PublishProblem    *problem.PublishProblemHandler
	HideProblem       *problem.HideProblemHandler
}

func NewProblemHandler(
	createProblem *problem.CreateProblemHandler,
	updateProblem *problem.UpdateProblemHandler,
	deleteProblem *problem.DeleteProblemHandler,
	getProblem *problem.GetProblemHandler,
	listProblems *problem.ListProblemsHandler,
	publishProblem *problem.PublishProblemHandler,
	hideProblem *problem.HideProblemHandler,
) *ProblemHandler {
	return &ProblemHandler{
		CreateProblem:     createProblem,
		UpdateProblem:     updateProblem,
		DeleteProblem:     deleteProblem,
		GetProblem:        getProblem,
		ListProblems:      listProblems,
		PublishProblem:    publishProblem,
		HideProblem:       hideProblem,
	}
}
