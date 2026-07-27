package problem

import (
	"context"
	"fmt"
	"strings"
	"time"

	problemv1 "go-judge-system/pkg/pb/problem/v1"
	"go-judge-system/workers/judge/internal/application/port/outbound"
	workerdomain "go-judge-system/workers/judge/internal/domain"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCMetadataReader struct {
	client  problemv1.ProblemServiceClient
	timeout time.Duration
	logger  *zap.Logger
}

func NewGRPCMetadataReader(client problemv1.ProblemServiceClient, timeout time.Duration, logger *zap.Logger) *GRPCMetadataReader {
	return &GRPCMetadataReader{
		client:  client,
		timeout: timeout,
		logger:  logger,
	}
}

func (r *GRPCMetadataReader) GetTestCaseMetadata(ctx context.Context, problemID int64) (outbound.ProblemTestCaseMetadata, error) {
	if problemID <= 0 {
		return outbound.ProblemTestCaseMetadata{}, workerdomain.MarkNonRetryable(
			fmt.Errorf("problem_id must be greater than zero"),
		)
	}

	if r.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.timeout)
		defer cancel()
	}

	resp, err := r.client.GetTestCase(ctx, &problemv1.GetTestCaseRequest{ProblemId: problemID})
	if err != nil {
		if isNonRetryableGRPCError(err) {
			return outbound.ProblemTestCaseMetadata{}, workerdomain.MarkNonRetryable(
				fmt.Errorf("get testcase metadata from problem-service: %w", err),
			)
		}
		return outbound.ProblemTestCaseMetadata{}, fmt.Errorf("get testcase metadata from problem-service: %w", err)
	}
	if resp == nil {
		return outbound.ProblemTestCaseMetadata{}, fmt.Errorf("problem-service returned nil testcase metadata")
	}

	metadata := outbound.ProblemTestCaseMetadata{
		ProblemID:      problemID,
		ZipDownloadURL: strings.TrimSpace(resp.GetZipDownloadUrl()),
		TestCount:      int(resp.GetTestCount()),
		Version:        int(resp.GetVersion()),
	}
	if metadata.ZipDownloadURL == "" {
		return outbound.ProblemTestCaseMetadata{}, workerdomain.MarkNonRetryable(
			fmt.Errorf("problem-service returned empty zip_download_url for problem_id=%d", problemID),
		)
	}
	if metadata.TestCount <= 0 {
		return outbound.ProblemTestCaseMetadata{}, workerdomain.MarkNonRetryable(
			fmt.Errorf("problem-service returned invalid test_count=%d for problem_id=%d", metadata.TestCount, problemID),
		)
	}
	if metadata.Version <= 0 {
		return outbound.ProblemTestCaseMetadata{}, workerdomain.MarkNonRetryable(
			fmt.Errorf("problem-service returned invalid version=%d for problem_id=%d", metadata.Version, problemID),
		)
	}

	r.logger.Debug(
		"loaded testcase metadata from problem-service gRPC",
		zap.Int64("problem_id", problemID),
		zap.Int("test_count", metadata.TestCount),
		zap.Int("version", metadata.Version),
	)
	return metadata, nil
}

func isNonRetryableGRPCError(err error) bool {
	switch status.Code(err) {
	case codes.InvalidArgument, codes.NotFound, codes.FailedPrecondition, codes.PermissionDenied:
		return true
	default:
		return false
	}
}
