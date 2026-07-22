package grpc

import (
	"context"

	problemv1 "go-judge-system/pkg/pb/problem/v1"
	"go-judge-system/services/problem/internal/adapter/inbound/grpc/handler"
)

// ProblemServer composes ProblemService handlers.
type ProblemServer struct {
	problemv1.UnimplementedProblemServiceServer

	worker  *handler.WorkerHandler
	problem *handler.ProblemHandler
}

var _ problemv1.ProblemServiceServer = (*ProblemServer)(nil)

func NewProblemServer(worker *handler.WorkerHandler, problem *handler.ProblemHandler) *ProblemServer {
	return &ProblemServer{worker: worker, problem: problem}
}

func (s *ProblemServer) GetProblem(
	ctx context.Context,
	req *problemv1.GetProblemRequest,
) (*problemv1.GetProblemResponse, error) {
	return s.problem.GetProblem.Handle(ctx, req)
}

func (s *ProblemServer) GetTestCase(
	ctx context.Context,
	req *problemv1.GetTestCaseRequest,
) (*problemv1.GetTestCaseResponse, error) {
	return s.worker.GetTestCase.Handle(ctx, req)
}
