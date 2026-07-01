package handler

import "go-judge-system/services/problem/internal/adapter/inbound/http/handler/admin/problem"

type AdminHandler struct {
	CreateProblem *problem.CreateProblemHandler
}

func NewAdminHandler(
	createProblem *problem.CreateProblemHandler,
) *AdminHandler {
	return &AdminHandler{
		CreateProblem: createProblem,
	}
}
