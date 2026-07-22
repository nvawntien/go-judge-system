package problem

import (
	"context"
	"strings"
	"time"

	problemv1 "go-judge-system/pkg/pb/problem/v1"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcProblemReader struct {
	client  problemv1.ProblemServiceClient
	timeout time.Duration
}

func NewGRPCProblemReader(
	client problemv1.ProblemServiceClient,
	timeout time.Duration,
) outbound.ProblemReader {
	return &grpcProblemReader{client: client, timeout: timeout}
}

func (r *grpcProblemReader) GetForSubmission(
	ctx context.Context,
	problemID int64,
) (outbound.ProblemForSubmission, error) {
	callCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	response, err := r.client.GetProblemForSubmission(
		callCtx,
		&problemv1.GetProblemForSubmissionRequest{ProblemId: problemID},
	)
	if err != nil {
		switch status.Code(err) {
		case codes.InvalidArgument:
			return outbound.ProblemForSubmission{}, domain.ErrInvalidProblemID
		case codes.NotFound:
			return outbound.ProblemForSubmission{}, domain.ErrProblemNotFound
		case codes.Canceled:
			if ctx.Err() != nil {
				return outbound.ProblemForSubmission{}, ctx.Err()
			}
			return outbound.ProblemForSubmission{}, context.Canceled
		case codes.DeadlineExceeded, codes.Unavailable, codes.Internal:
			return outbound.ProblemForSubmission{}, domain.ErrProblemServiceUnavailable.Wrap(err)
		default:
			return outbound.ProblemForSubmission{}, domain.ErrProblemServiceUnavailable.Wrap(err)
		}
	}
	if response == nil || response.GetProblemId() <= 0 || strings.TrimSpace(response.GetTitle()) == "" || strings.TrimSpace(response.GetSlug()) == "" {
		return outbound.ProblemForSubmission{}, domain.ErrProblemServiceUnavailable
	}

	return outbound.ProblemForSubmission{
		ID:    response.GetProblemId(),
		Title: response.GetTitle(),
		Slug:  response.GetSlug(),
	}, nil
}
