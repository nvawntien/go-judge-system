package grpc

import (
	"context"

	problemv1 "go-judge-system/pkg/pb/problem/v1"
	"go-judge-system/services/problem/internal/adapter/inbound/grpc/handler"
)

// ProblemServer composes actor-specific handlers into ProblemService.
type ProblemServer struct {
	problemv1.UnimplementedProblemServiceServer

	worker *handler.WorkerHandler
}

var _ problemv1.ProblemServiceServer = (*ProblemServer)(nil)

func NewProblemServer(worker *handler.WorkerHandler) *ProblemServer {
	return &ProblemServer{worker: worker}
}

func (s *ProblemServer) GetTestCase(
	ctx context.Context,
	req *problemv1.GetTestCaseRequest,
) (*problemv1.GetTestCaseResponse, error) {
	return s.worker.GetTestCase.Handle(ctx, req)
}
