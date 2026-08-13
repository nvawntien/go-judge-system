package grpc

import (
	"fmt"
	"net"

	"go-judge-system/pkg/config"
	sharedgrpc "go-judge-system/pkg/grpc"
	authv1 "go-judge-system/pkg/pb/auth/v1"

	googlegrpc "google.golang.org/grpc"
)

type Server struct {
	server  *googlegrpc.Server
	address string
}

func NewServer(cfg config.ServerConfig, publicUserServer *PublicUserServer) *Server {
	grpcServer := sharedgrpc.NewServer()
	authv1.RegisterPublicUserServiceServer(grpcServer, publicUserServer)
	return &Server{server: grpcServer, address: fmt.Sprintf(":%d", cfg.GRPCPort)}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("listen for gRPC on %s: %w", s.address, err)
	}
	if err := s.server.Serve(listener); err != nil {
		return fmt.Errorf("serve gRPC on %s: %w", s.address, err)
	}
	return nil
}

func (s *Server) Stop() {
	s.server.GracefulStop()
}
