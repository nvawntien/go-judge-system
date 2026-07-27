package problem

import (
	"context"
	"errors"
	"testing"
	"time"

	problemv1 "go-judge-system/pkg/pb/problem/v1"
	workerdomain "go-judge-system/workers/judge/internal/domain"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeProblemServiceClient struct {
	resp *problemv1.GetTestCaseResponse
	err  error
}

func (f fakeProblemServiceClient) GetTestCase(
	context.Context,
	*problemv1.GetTestCaseRequest,
	...grpc.CallOption,
) (*problemv1.GetTestCaseResponse, error) {
	return f.resp, f.err
}

func (f fakeProblemServiceClient) GetProblem(
	context.Context,
	*problemv1.GetProblemRequest,
	...grpc.CallOption,
) (*problemv1.GetProblemResponse, error) {
	return nil, errors.New("not implemented")
}

func TestGRPCMetadataReaderGetTestCaseMetadata(t *testing.T) {
	t.Parallel()

	reader := NewGRPCMetadataReader(fakeProblemServiceClient{
		resp: &problemv1.GetTestCaseResponse{
			ZipDownloadUrl: " http://minio/testcases.zip ",
			TestCount:      3,
			Version:        7,
		},
	}, time.Second, zap.NewNop())

	got, err := reader.GetTestCaseMetadata(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetTestCaseMetadata() error = %v", err)
	}
	if got.ProblemID != 42 || got.ZipDownloadURL != "http://minio/testcases.zip" || got.TestCount != 3 || got.Version != 7 {
		t.Fatalf("metadata = %#v", got)
	}
}

func TestGRPCMetadataReaderMarksNotFoundNonRetryable(t *testing.T) {
	t.Parallel()

	reader := NewGRPCMetadataReader(fakeProblemServiceClient{
		err: status.Error(codes.NotFound, "missing"),
	}, time.Second, zap.NewNop())

	_, err := reader.GetTestCaseMetadata(context.Background(), 42)
	if err == nil {
		t.Fatal("GetTestCaseMetadata() error = nil")
	}
	if !workerdomain.IsNonRetryable(err) {
		t.Fatalf("IsNonRetryable(%v) = false, want true", err)
	}
}
