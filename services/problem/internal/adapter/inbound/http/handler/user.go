package handler

import (
	userproblem "go-judge-system/services/problem/internal/adapter/inbound/http/handler/user/problem"
	usertag "go-judge-system/services/problem/internal/adapter/inbound/http/handler/user/tag"
)

type UserHandler struct {
	ListProblems *userproblem.ListProblemsHandler
	GetProblem   *userproblem.GetProblemHandler
	ListTags     *usertag.ListTagsHandler
}

func NewUserHandler(
	listProblems *userproblem.ListProblemsHandler,
	getProblem *userproblem.GetProblemHandler,
	listTags *usertag.ListTagsHandler,
) *UserHandler {
	return &UserHandler{
		ListProblems: listProblems,
		GetProblem:   getProblem,
		ListTags:     listTags,
	}
}
