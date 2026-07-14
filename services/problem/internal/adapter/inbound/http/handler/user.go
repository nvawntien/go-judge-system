package handler

import (
	userproblem "go-judge-system/services/problem/internal/adapter/inbound/http/handler/user/problem"
	usertag "go-judge-system/services/problem/internal/adapter/inbound/http/handler/user/tag"
)

type UserHandler struct {
	ListProblems   *userproblem.ListProblemsHandler
	ListMyProblems *userproblem.ListMyProblemsHandler
	GetProblem     *userproblem.GetProblemHandler
	ListTags       *usertag.ListTagsHandler
}

func NewUserHandler(
	listProblems *userproblem.ListProblemsHandler,
	listMyProblems *userproblem.ListMyProblemsHandler,
	getProblem *userproblem.GetProblemHandler,
	listTags *usertag.ListTagsHandler,
) *UserHandler {
	return &UserHandler{
		ListProblems:   listProblems,
		ListMyProblems: listMyProblems,
		GetProblem:     getProblem,
		ListTags:       listTags,
	}
}
