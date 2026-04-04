package handler

import testcase "go-judge-system/services/problem/internal/adapter/inbound/http/handler/test_case"

type TestCaseHandler struct {
	UploadTestCase         *testcase.UploadTestCaseHandler
	GetTestCaseForWorker   *testcase.GetTestCaseForWorkerHandler
}

func NewTestCaseHandler(
	uploadTestCase *testcase.UploadTestCaseHandler,
	getTestCaseForWorker *testcase.GetTestCaseForWorkerHandler,
) *TestCaseHandler {
	return &TestCaseHandler{
		UploadTestCase:       uploadTestCase,
		GetTestCaseForWorker: getTestCaseForWorker,
	}
}
