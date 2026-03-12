package handler

import testcase "go-judge-system/services/problem/internal/adapter/inbound/http/handler/test_case"

type TestCaseHandler struct {
	CreateTestCase *testcase.CreateTestCaseHandler
	ListTestCases  *testcase.ListTestCasesHandler
	UpdateTestCase *testcase.UpdateTestCaseHandler
	DeleteTestCase *testcase.DeleteTestCaseHandler
}

func NewTestCaseHandler(
	createTestCase *testcase.CreateTestCaseHandler,
	listTestCases *testcase.ListTestCasesHandler,
	updateTestCase *testcase.UpdateTestCaseHandler,
	deleteTestCase *testcase.DeleteTestCaseHandler,
) *TestCaseHandler {
	return &TestCaseHandler{
		CreateTestCase: createTestCase,
		ListTestCases:  listTestCases,
		UpdateTestCase: updateTestCase,
		DeleteTestCase: deleteTestCase,
	}
}
