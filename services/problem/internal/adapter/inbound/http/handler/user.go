package handler

import userproblem "go-judge-system/services/problem/internal/adapter/inbound/http/handler/user/problem"

type UserHandler struct {
	ListProblems *userproblem.ListProblemsHandler
	GetProblem   *userproblem.GetProblemHandler
}

func NewUserHandler(
	listProblems *userproblem.ListProblemsHandler,
	getProblem *userproblem.GetProblemHandler,
) *UserHandler {
	return &UserHandler{
		ListProblems: listProblems,
		GetProblem:   getProblem,
	}
}
