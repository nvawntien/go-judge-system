package handler

import testcase "go-judge-system/services/problem/internal/adapter/inbound/http/handler/test_case"

type TestCaseHandler struct {
	UploadTestCase *testcase.UploadTestCaseHandler
}

func NewTestCaseHandler(
	uploadTestCase *testcase.UploadTestCaseHandler,
) *TestCaseHandler {
	return &TestCaseHandler{
		UploadTestCase: uploadTestCase,
	}
}
