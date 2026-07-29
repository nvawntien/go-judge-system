package problem

import (
	"context"
	"errors"
	"strings"
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
	seen *problemv1.GetTestCaseRequest
}

func (f *fakeProblemServiceClient) GetTestCase(
	_ context.Context,
	req *problemv1.GetTestCaseRequest,
	_ ...grpc.CallOption,
) (*problemv1.GetTestCaseResponse, error) {
	f.seen = req
	return f.resp, f.err
}

func (f *fakeProblemServiceClient) GetProblem(
	_ context.Context,
	_ *problemv1.GetProblemRequest,
	_ ...grpc.CallOption,
) (*problemv1.GetProblemResponse, error) {
	return nil, errors.New("not implemented")
}

func TestGRPCMetadataReaderGetTestCaseMetadata(t *testing.T) {
	t.Parallel()

	client := &fakeProblemServiceClient{
		resp: &problemv1.GetTestCaseResponse{
			ZipDownloadUrl: " http://minio/testcases.zip ",
			TestCount:      3,
			Version:        7,
		},
	}
	reader := NewGRPCMetadataReader(client, time.Second, zap.NewNop())

	got, err := reader.GetTestCaseMetadata(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetTestCaseMetadata() error = %v", err)
	}
	if client.seen == nil || client.seen.GetProblemId() != 42 {
		t.Fatalf("GetTestCase request = %+v, want problem_id=42", client.seen)
	}
	if got.ProblemID != 42 || got.ZipDownloadURL != "http://minio/testcases.zip" || got.TestCount != 3 || got.Version != 7 {
		t.Fatalf("metadata = %#v", got)
	}
}

func TestGRPCMetadataReaderMarksNotFoundNonRetryable(t *testing.T) {
	t.Parallel()

	reader := NewGRPCMetadataReader(&fakeProblemServiceClient{
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

func TestGRPCMetadataReaderRejectsInvalidProblemServiceResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response *problemv1.GetTestCaseResponse
		wantErr  string
	}{
		{
			name:     "nil response",
			response: nil,
			wantErr:  "nil testcase metadata",
		},
		{
			name: "empty URL",
			response: &problemv1.GetTestCaseResponse{
				TestCount: 24,
				Version:   1,
			},
			wantErr: "empty zip_download_url",
		},
		{
			name: "invalid test count",
			response: &problemv1.GetTestCaseResponse{
				ZipDownloadUrl: "http://minio.local/testcases.zip",
				Version:        1,
			},
			wantErr: "invalid test_count",
		},
		{
			name: "invalid version",
			response: &problemv1.GetTestCaseResponse{
				ZipDownloadUrl: "http://minio.local/testcases.zip",
				TestCount:      24,
			},
			wantErr: "invalid version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reader := NewGRPCMetadataReader(&fakeProblemServiceClient{resp: tt.response}, time.Second, zap.NewNop())

			_, err := reader.GetTestCaseMetadata(context.Background(), 42)
			if err == nil {
				t.Fatal("GetTestCaseMetadata() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("GetTestCaseMetadata() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestGRPCMetadataReaderPropagatesRetryableGRPCError(t *testing.T) {
	t.Parallel()

	reader := NewGRPCMetadataReader(&fakeProblemServiceClient{
		err: status.Error(codes.Unavailable, "problem unavailable"),
	}, time.Second, zap.NewNop())

	_, err := reader.GetTestCaseMetadata(context.Background(), 42)
	if err == nil {
		t.Fatal("GetTestCaseMetadata() error = nil, want error")
	}
	if workerdomain.IsNonRetryable(err) {
		t.Fatalf("IsNonRetryable(%v) = true, want false", err)
	}
	if !strings.Contains(err.Error(), "get testcase metadata from problem-service") {
		t.Fatalf("GetTestCaseMetadata() error = %v, want gRPC context", err)
	}
}
