package handler

import (
	"context"
	"errors"
	"strings"

	authv1 "go-judge-system/pkg/pb/auth/v1"
	"go-judge-system/services/auth/internal/application/dto"
	inbound "go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ResolvePublicUserHandler struct {
	useCase inbound.ResolvePublicUserUseCase
}

func NewResolvePublicUserHandler(useCase inbound.ResolvePublicUserUseCase) *ResolvePublicUserHandler {
	return &ResolvePublicUserHandler{useCase: useCase}
}

func (h *ResolvePublicUserHandler) Handle(
	ctx context.Context,
	req *authv1.ResolvePublicUserRequest,
) (*authv1.ResolvePublicUserResponse, error) {
	if req == nil || strings.TrimSpace(req.GetUsername()) == "" {
		return nil, status.Error(codes.InvalidArgument, "username is required")
	}

	result, err := h.useCase.Execute(ctx, dto.ResolvePublicUserRequest{Username: req.GetUsername()})
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			return nil, status.Error(codes.Canceled, "request canceled")
		case errors.Is(err, context.DeadlineExceeded):
			return nil, status.Error(codes.DeadlineExceeded, "request deadline exceeded")
		case errors.Is(err, domain.ErrUserNotFound):
			return nil, status.Error(codes.NotFound, "public user not found")
		default:
			return nil, status.Error(codes.Internal, "failed to resolve public user")
		}
	}

	return &authv1.ResolvePublicUserResponse{UserId: result.UserID, Username: result.Username}, nil
}
