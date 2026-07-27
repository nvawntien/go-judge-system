package problem

import (
	"context"
	"strings"
	"time"

	problemv1 "go-judge-system/pkg/pb/problem/v1"
	"go-judge-system/services/submission/internal/application/dto"
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
	actor dto.ProblemActor,
) (dto.ProblemMetadata, error) {
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
			return dto.ProblemMetadata{}, domain.ErrInvalidProblemID
		case codes.Unauthenticated:
			return dto.ProblemMetadata{}, domain.ErrProblemActorUnauthenticated
		case codes.PermissionDenied:
			return dto.ProblemMetadata{}, domain.ErrProblemActorForbidden
		case codes.NotFound:
			return dto.ProblemMetadata{}, domain.ErrProblemNotFound
		case codes.Canceled:
			if ctx.Err() != nil {
				return dto.ProblemMetadata{}, ctx.Err()
			}
			return dto.ProblemMetadata{}, context.Canceled
		case codes.DeadlineExceeded, codes.Unavailable, codes.Internal:
			return dto.ProblemMetadata{}, domain.ErrProblemServiceUnavailable.Wrap(err)
		default:
			return dto.ProblemMetadata{}, domain.ErrProblemServiceUnavailable.Wrap(err)
		}
	}
	if response == nil || response.GetProblemId() <= 0 || strings.TrimSpace(response.GetTitle()) == "" || strings.TrimSpace(response.GetSlug()) == "" {
		return dto.ProblemMetadata{}, domain.ErrProblemServiceUnavailable
	}

	return dto.ProblemMetadata{
		ID:          response.GetProblemId(),
		Title:       response.GetTitle(),
		Slug:        response.GetSlug(),
		TimeLimit:   response.GetTimeLimit(),
		MemoryLimit: int(response.GetMemoryLimit()),
	}, nil
}
