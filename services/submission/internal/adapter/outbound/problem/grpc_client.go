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

func (r *grpcProblemReader) GetProblem(
	ctx context.Context,
	problemID int64,
	actor outbound.ProblemActor,
) (outbound.ProblemMetadata, error) {
	callCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	response, err := r.client.GetProblem(
		callCtx,
		&problemv1.GetProblemRequest{
			ProblemId:   problemID,
			ActorUserId: actor.UserID,
			ActorRole:   string(actor.Role),
		},
	)
	if err != nil {
		switch status.Code(err) {
		case codes.InvalidArgument:
			return outbound.ProblemMetadata{}, domain.ErrInvalidProblemID
		case codes.Unauthenticated:
			return outbound.ProblemMetadata{}, domain.ErrProblemActorUnauthenticated
		case codes.PermissionDenied:
			return outbound.ProblemMetadata{}, domain.ErrProblemActorForbidden
		case codes.NotFound:
			return outbound.ProblemMetadata{}, domain.ErrProblemNotFound
		case codes.Canceled:
			if ctx.Err() != nil {
				return outbound.ProblemMetadata{}, ctx.Err()
			}
			return outbound.ProblemMetadata{}, context.Canceled
		case codes.DeadlineExceeded, codes.Unavailable, codes.Internal:
			return outbound.ProblemMetadata{}, domain.ErrProblemServiceUnavailable.Wrap(err)
		default:
			return outbound.ProblemMetadata{}, domain.ErrProblemServiceUnavailable.Wrap(err)
		}
	}
	if response == nil || response.GetProblemId() <= 0 || strings.TrimSpace(response.GetTitle()) == "" || strings.TrimSpace(response.GetSlug()) == "" {
		return outbound.ProblemMetadata{}, domain.ErrProblemServiceUnavailable
	}

	return outbound.ProblemMetadata{
		ID:    response.GetProblemId(),
		Title: response.GetTitle(),
		Slug:  response.GetSlug(),
	}, nil
}
