package grpc

import (
	"context"
	"testing"

	"go-judge-system/pkg/config"
	problemv1 "go-judge-system/pkg/pb/problem/v1"
	workertestcase"go-judge-system/services/problem/internal/adapter/inbound/grpc/handler/worker/testcase"
	"go-judge-system/services/problem/internal/application/dto"
	"go-judge-system/services/problem/internal/adapter/inbound/grpc/handler"
)

type stubGetTestCaseUseCase struct {
	result    *dto.InternalTestCaseResponse
	problemID int64
}

func (s *stubGetTestCaseUseCase) Execute(
	_ context.Context,
	problemID int64,
) (*dto.InternalTestCaseResponse, error) {
	s.problemID = problemID
	return s.result, nil
}

func TestProblemServerDelegatesGetTestCaseToWorkerHandler(t *testing.T) {
	t.Parallel()

	useCase := &stubGetTestCaseUseCase{
		result: &dto.InternalTestCaseResponse{
			ZipDownloadURL: "https://storage.internal/testcases.zip",
			TestCount:      12,
			Version:        3,
		},
	}
	server := NewProblemServer(handler.NewWorkerHandler(workertestcase.NewGetTestCaseHandler(useCase)))

	response, err := server.GetTestCase(context.Background(), &problemv1.GetTestCaseRequest{ProblemId: 42})
	if err != nil {
		t.Fatalf("GetTestCase() error = %v", err)
	}
	if useCase.problemID != 42 {
		t.Fatalf("delegated problem ID = %d, want 42", useCase.problemID)
	}
	if response.GetZipDownloadUrl() != useCase.result.ZipDownloadURL {
		t.Fatalf("zip download URL = %q, want %q", response.GetZipDownloadUrl(), useCase.result.ZipDownloadURL)
	}
}

func TestNewServerRegistersProblemService(t *testing.T) {
	t.Parallel()

	getTestCaseHandler := workertestcase.NewGetTestCaseHandler(&stubGetTestCaseUseCase{
		result: &dto.InternalTestCaseResponse{},
	})
	problemServer := NewProblemServer(handler.NewWorkerHandler(getTestCaseHandler))
	server := NewServer(config.ServerConfig{GRPCPort: 9092}, problemServer)

	if server.address != ":9092" {
		t.Fatalf("server address = %q, want %q", server.address, ":9092")
	}
	if _, ok := server.server.GetServiceInfo()["problem.v1.ProblemService"]; !ok {
		t.Fatal("ProblemService was not registered")
	}
}
