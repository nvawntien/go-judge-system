package grpc

import (
	"context"

	judgev1 "go-judge-system/pkg/pb/judge/v1"
	"go-judge-system/workers/judge/internal/adapter/inbound/grpc/handler"
)

type JudgeServer struct {
	judgev1.UnimplementedJudgeServiceServer

	runCode *handler.RunCodeHandler
}

var _ judgev1.JudgeServiceServer = (*JudgeServer)(nil)

func NewJudgeServer(runCode *handler.RunCodeHandler) *JudgeServer {
	return &JudgeServer{runCode: runCode}
}

func (s *JudgeServer) RunCode(ctx context.Context, req *judgev1.RunCodeRequest) (*judgev1.RunCodeResponse, error) {
	return s.runCode.Handle(ctx, req)
}
