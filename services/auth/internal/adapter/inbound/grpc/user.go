package grpc

import (
	"context"

	authv1 "go-judge-system/pkg/pb/auth/v1"
	"go-judge-system/services/auth/internal/adapter/inbound/grpc/handler"
)

type PublicUserServer struct {
	authv1.UnimplementedPublicUserServiceServer
	resolvePublicUser *handler.ResolvePublicUserHandler
}

var _ authv1.PublicUserServiceServer = (*PublicUserServer)(nil)

func NewPublicUserServer(resolvePublicUser *handler.ResolvePublicUserHandler) *PublicUserServer {
	return &PublicUserServer{resolvePublicUser: resolvePublicUser}
}

func (s *PublicUserServer) ResolvePublicUser(
	ctx context.Context,
	req *authv1.ResolvePublicUserRequest,
) (*authv1.ResolvePublicUserResponse, error) {
	return s.resolvePublicUser.Handle(ctx, req)
}
