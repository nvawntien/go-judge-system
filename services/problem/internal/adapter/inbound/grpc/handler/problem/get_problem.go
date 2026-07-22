package problem

import (
	"context"
	"errors"

	problemv1 "go-judge-system/pkg/pb/problem/v1"
	"go-judge-system/pkg/rbac"
	inbound "go-judge-system/services/problem/internal/application/port/inbound"
	"go-judge-system/services/problem/internal/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetProblemHandler struct {
	useCase inbound.GetProblemUseCase
}

func NewGetProblemHandler(useCase inbound.GetProblemUseCase) *GetProblemHandler {
	return &GetProblemHandler{useCase: useCase}
}

func (h *GetProblemHandler) Handle(
	ctx context.Context,
	req *problemv1.GetProblemRequest,
) (*problemv1.GetProblemResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	result, err := h.useCase.Execute(ctx, inbound.GetProblemRequest{
		ProblemID:   req.GetProblemId(),
		ActorUserID: req.GetActorUserId(),
		ActorRole:   rbac.Role(req.GetActorRole()),
	})
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			return nil, status.Error(codes.Canceled, "request canceled")
		case errors.Is(err, context.DeadlineExceeded):
			return nil, status.Error(codes.DeadlineExceeded, "request deadline exceeded")
		case errors.Is(err, domain.ErrInvalidInput):
			return nil, status.Error(codes.InvalidArgument, "problem_id must be greater than zero")
		case errors.Is(err, domain.ErrActorUnauthenticated):
			return nil, status.Error(codes.Unauthenticated, "actor identity and role are required")
		case errors.Is(err, domain.ErrPermissionDenied):
			return nil, status.Error(codes.PermissionDenied, "unsupported actor role")
		case errors.Is(err, domain.ErrProblemNotFound):
			return nil, status.Error(codes.NotFound, "problem not found")
		default:
			return nil, status.Error(codes.Internal, "failed to get problem")
		}
	}

	return &problemv1.GetProblemResponse{
		ProblemId: result.ID,
		Title:     result.Title,
		Slug:      result.Slug,
	}, nil
}
