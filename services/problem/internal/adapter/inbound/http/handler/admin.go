package handler

import (
	"go-judge-system/services/problem/internal/adapter/inbound/http/handler/admin/problem"
	"go-judge-system/services/problem/internal/adapter/inbound/http/handler/admin/testcase"
)

type AdminHandler struct {
	// Problem management
	CreateProblem *problem.CreateProblemHandler
	// Test case management
	UploadTestCase *testcase.UploadTestCaseHandler
}

func NewAdminHandler(
	// Problem management
	createProblem *problem.CreateProblemHandler,
	// Test case management
	uploadTestCase *testcase.UploadTestCaseHandler,
) *AdminHandler {
	return &AdminHandler{
		// Problem management
		CreateProblem:  createProblem,
		// Test case management
		UploadTestCase: uploadTestCase,
	}
}
