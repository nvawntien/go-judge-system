package handler

import "go-judge-system/services/problem/internal/adapter/inbound/grpc/handler/worker/testcase"

// WorkerHandler groups the gRPC handlers exposed to judge workers.
type WorkerHandler struct {
	GetTestCase *testcase.GetTestCaseHandler
}

func NewWorkerHandler(getTestCase *testcase.GetTestCaseHandler) *WorkerHandler {
	return &WorkerHandler{GetTestCase: getTestCase}
}
