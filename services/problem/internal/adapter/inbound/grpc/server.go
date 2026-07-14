package grpc

import (
	"fmt"
	"net"

	"go-judge-system/pkg/config"
	sharedgrpc "go-judge-system/pkg/grpc"
	problemv1 "go-judge-system/pkg/pb/problem/v1"

	googlegrpc "google.golang.org/grpc"
)

type Server struct {
	server  *googlegrpc.Server
	address string
}

func NewServer(cfg config.ServerConfig, problemServer *ProblemServer) *Server {
	grpcServer := sharedgrpc.NewServer()
	problemv1.RegisterProblemServiceServer(grpcServer, problemServer)

	return &Server{
		server:  grpcServer,
		address: fmt.Sprintf(":%d", cfg.GRPCPort),
	}
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
