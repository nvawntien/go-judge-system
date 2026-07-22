package problem

import (
	"context"
	"errors"

	problemv1 "go-judge-system/pkg/pb/problem/v1"
	submissioninbound "go-judge-system/services/problem/internal/application/port/inbound/submission"
	"go-judge-system/services/problem/internal/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetProblemForSubmissionHandler struct {
	useCase submissioninbound.GetProblemForSubmissionUseCase
}

func NewGetProblemForSubmissionHandler(
	useCase submissioninbound.GetProblemForSubmissionUseCase,
) *GetProblemForSubmissionHandler {
	return &GetProblemForSubmissionHandler{useCase: useCase}
}

func (h *GetProblemForSubmissionHandler) Handle(
	ctx context.Context,
	req *problemv1.GetProblemForSubmissionRequest,
) (*problemv1.GetProblemForSubmissionResponse, error) {
	if req == nil || req.GetProblemId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "problem_id must be greater than zero")
	}

	result, err := h.useCase.Execute(ctx, req.GetProblemId())
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			return nil, status.Error(codes.Canceled, "request canceled")
		case errors.Is(err, context.DeadlineExceeded):
			return nil, status.Error(codes.DeadlineExceeded, "request deadline exceeded")
		case errors.Is(err, domain.ErrInvalidInput):
			return nil, status.Error(codes.InvalidArgument, "problem_id must be greater than zero")
		case errors.Is(err, domain.ErrProblemNotFound):
			return nil, status.Error(codes.NotFound, "problem not found")
		default:
			return nil, status.Error(codes.Internal, "failed to get problem")
		}
	}

	return &problemv1.GetProblemForSubmissionResponse{
		ProblemId: result.ProblemID,
		Title:     result.Title,
		Slug:      result.Slug,
	}, nil
}
