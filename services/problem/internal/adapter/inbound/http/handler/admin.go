package handler

import (
	"go-judge-system/services/problem/internal/adapter/inbound/http/handler/admin/problem"
	"go-judge-system/services/problem/internal/adapter/inbound/http/handler/admin/tag"
	"go-judge-system/services/problem/internal/adapter/inbound/http/handler/admin/testcase"
)

type AdminHandler struct {
	// Problem management
	CreateProblem  *problem.CreateProblemHandler
	ListProblems   *problem.ListProblemsHandler
	UpdateProblem  *problem.UpdateProblemHandler
	GetProblem     *problem.GetProblemHandler
	PublishProblem *problem.PublishProblemHandler
	HiddenProblem  *problem.HiddenProblemHandler
	DeleteProblem  *problem.DeleteProblemHandler
	// Tag management
	ListTags  *tag.ListTagsHandler
	CreateTag *tag.CreateTagHandler
	UpdateTag *tag.UpdateTagHandler
	DeleteTag *tag.DeleteTagHandler
	// Test case management
	GetTestCase    *testcase.GetTestCaseHandler
	UploadTestCase *testcase.UploadTestCaseHandler
}

func NewAdminHandler(
	// Problem management
	createProblem *problem.CreateProblemHandler,
	listProblems *problem.ListProblemsHandler,
	updateProblem *problem.UpdateProblemHandler,
	getProblem *problem.GetProblemHandler,
	publishProblem *problem.PublishProblemHandler,
	hiddenProblem *problem.HiddenProblemHandler,
	deleteProblem *problem.DeleteProblemHandler,
	// Tag management
	listTags *tag.ListTagsHandler,
	createTag *tag.CreateTagHandler,
	updateTag *tag.UpdateTagHandler,
	deleteTag *tag.DeleteTagHandler,
	// Test case management
	getTestCase *testcase.GetTestCaseHandler,
	uploadTestCase *testcase.UploadTestCaseHandler,
) *AdminHandler {
	return &AdminHandler{
		// Problem management
		CreateProblem:  createProblem,
		ListProblems:   listProblems,
		UpdateProblem:  updateProblem,
		GetProblem:     getProblem,
		PublishProblem: publishProblem,
		HiddenProblem:  hiddenProblem,
		DeleteProblem:  deleteProblem,
		// Tag management
		ListTags:  listTags,
		CreateTag: createTag,
		UpdateTag: updateTag,
		DeleteTag: deleteTag,
		// Test case management
		GetTestCase:    getTestCase,
		UploadTestCase: uploadTestCase,
	}
}
