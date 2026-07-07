package handler

import (
	"go-judge-system/services/problem/internal/adapter/inbound/http/handler/admin/problem"
	"go-judge-system/services/problem/internal/adapter/inbound/http/handler/admin/testcase"
)

type AdminHandler struct {
	// Problem management
	CreateProblem  *problem.CreateProblemHandler
	ListProblems   *problem.ListProblemsHandler
	GetProblem     *problem.GetProblemHandler
	PublishProblem *problem.PublishProblemHandler
	HiddenProblem  *problem.HiddenProblemHandler
	// Test case management
	UploadTestCase *testcase.UploadTestCaseHandler
}

func NewAdminHandler(
	// Problem management
	createProblem *problem.CreateProblemHandler,
	listProblems *problem.ListProblemsHandler,
	getProblem *problem.GetProblemHandler,
	publishProblem *problem.PublishProblemHandler,
	hiddenProblem *problem.HiddenProblemHandler,
	// Test case management
	uploadTestCase *testcase.UploadTestCaseHandler,
) *AdminHandler {
	return &AdminHandler{
		// Problem management
		CreateProblem:  createProblem,
		ListProblems:   listProblems,
		GetProblem:     getProblem,
		PublishProblem: publishProblem,
		HiddenProblem:  hiddenProblem,
		// Test case management
		UploadTestCase: uploadTestCase,
	}
}
