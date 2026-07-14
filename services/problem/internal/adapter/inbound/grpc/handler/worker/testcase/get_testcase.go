package testcase

import (
	"context"
	"errors"

	problemv1 "go-judge-system/pkg/pb/problem/v1"
	workerinbound "go-judge-system/services/problem/internal/application/port/inbound/worker"
	"go-judge-system/services/problem/internal/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetTestCaseHandler struct {
	useCase workerinbound.GetTestCaseUseCase
}

func NewGetTestCaseHandler(useCase workerinbound.GetTestCaseUseCase) *GetTestCaseHandler {
	return &GetTestCaseHandler{
		useCase: useCase,
	}
}

func (h *GetTestCaseHandler) Handle(
	ctx context.Context,
	req *problemv1.GetTestCaseRequest,
) (*problemv1.GetTestCaseResponse, error) {
	if req == nil || req.GetProblemId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "problem_id must be greater than zero")
	}

	result, err := h.useCase.Execute(ctx, req.GetProblemId())
	if err != nil {
		if errors.Is(err, domain.ErrTestCaseNotFound) {
			return nil, status.Error(codes.NotFound, "test case not found")
		}

		return nil, status.Error(codes.Internal, "failed to get test case")
	}
	if result == nil {
		return nil, status.Error(codes.Internal, "failed to get test case")
	}

	return &problemv1.GetTestCaseResponse{
		ZipDownloadUrl: result.ZipDownloadURL,
		TestCount:      int32(result.TestCount),
		Version:        int32(result.Version),
	}, nil
}
