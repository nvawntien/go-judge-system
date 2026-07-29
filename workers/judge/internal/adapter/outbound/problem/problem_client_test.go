package problem

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	problemv1 "go-judge-system/pkg/pb/problem/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type fakeProblemServiceClient struct {
	response *problemv1.GetTestCaseResponse
	err      error
	seen     *problemv1.GetTestCaseRequest
}

func (f *fakeProblemServiceClient) GetTestCase(
	ctx context.Context,
	req *problemv1.GetTestCaseRequest,
	_ ...grpc.CallOption,
) (*problemv1.GetTestCaseResponse, error) {
	f.seen = req
	if f.err != nil {
		return nil, f.err
	}
	return f.response, nil
}

func (f *fakeProblemServiceClient) GetProblem(
	context.Context,
	*problemv1.GetProblemRequest,
	...grpc.CallOption,
) (*problemv1.GetProblemResponse, error) {
	return nil, errors.New("unexpected GetProblem call")
}

func TestFetchMetadataUsesProblemServiceGRPC(t *testing.T) {
	fake := &fakeProblemServiceClient{
		response: &problemv1.GetTestCaseResponse{
			ZipDownloadUrl: "http://minio.local/testcases.zip",
			TestCount:      24,
			Version:        3,
		},
	}
	client := NewProblemServiceClient(fake, time.Second, zap.NewNop())

	meta, err := client.fetchMetadata(context.Background(), 42)
	if err != nil {
		t.Fatalf("fetchMetadata() error = %v", err)
	}

	if fake.seen == nil || fake.seen.GetProblemId() != 42 {
		t.Fatalf("GetTestCase request = %+v, want problem_id=42", fake.seen)
	}
	if meta.TestCount != 24 || meta.Version != "3" || meta.ZipDownloadURL != "http://minio.local/testcases.zip" {
		t.Fatalf("metadata = %+v", meta)
	}
}

func TestFetchMetadataRejectsInvalidProblemServiceResponse(t *testing.T) {
	tests := []struct {
		name     string
		response *problemv1.GetTestCaseResponse
		wantErr  string
	}{
		{
			name:     "nil response",
			response: nil,
			wantErr:  "empty metadata",
		},
		{
			name: "empty URL",
			response: &problemv1.GetTestCaseResponse{
				TestCount: 24,
				Version:   1,
			},
			wantErr: "empty zip download URL",
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
			client := NewProblemServiceClient(&fakeProblemServiceClient{response: tt.response}, time.Second, zap.NewNop())

			_, err := client.fetchMetadata(context.Background(), 42)
			if err == nil {
				t.Fatal("fetchMetadata() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("fetchMetadata() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestFetchMetadataPropagatesProblemServiceGRPCError(t *testing.T) {
	client := NewProblemServiceClient(&fakeProblemServiceClient{err: context.DeadlineExceeded}, time.Second, zap.NewNop())

	_, err := client.fetchMetadata(context.Background(), 42)
	if err == nil {
		t.Fatal("fetchMetadata() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "call problem service gRPC") {
		t.Fatalf("fetchMetadata() error = %v, want gRPC context", err)
	}
}
