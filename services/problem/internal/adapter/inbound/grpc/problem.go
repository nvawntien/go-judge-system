package grpc

import (
	"context"

	problemv1 "go-judge-system/pkg/pb/problem/v1"
	"go-judge-system/services/problem/internal/adapter/inbound/grpc/handler"
)

// ProblemServer composes actor-specific handlers into ProblemService.
type ProblemServer struct {
	problemv1.UnimplementedProblemServiceServer

	worker     *handler.WorkerHandler
	submission *handler.SubmissionHandler
}

var _ problemv1.ProblemServiceServer = (*ProblemServer)(nil)

func NewProblemServer(worker *handler.WorkerHandler, submission *handler.SubmissionHandler) *ProblemServer {
	return &ProblemServer{worker: worker, submission: submission}
}

func (s *ProblemServer) GetProblemForSubmission(
	ctx context.Context,
	req *problemv1.GetProblemForSubmissionRequest,
) (*problemv1.GetProblemForSubmissionResponse, error) {
	return s.submission.GetProblemForSubmission.Handle(ctx, req)
}

func (s *ProblemServer) GetTestCase(
	ctx context.Context,
	req *problemv1.GetTestCaseRequest,
) (*problemv1.GetTestCaseResponse, error) {
	return s.worker.GetTestCase.Handle(ctx, req)
}
