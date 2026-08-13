package auth

import (
	"context"
	"strings"
	"time"

	authv1 "go-judge-system/pkg/pb/auth/v1"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcPublicUserResolver struct {
	client  authv1.PublicUserServiceClient
	timeout time.Duration
}

func NewGRPCPublicUserResolver(
	client authv1.PublicUserServiceClient,
	timeout time.Duration,
) outbound.PublicUserResolver {
	return &grpcPublicUserResolver{client: client, timeout: timeout}
}

func (r *grpcPublicUserResolver) ResolvePublicUser(ctx context.Context, username string) (outbound.PublicUser, error) {
	callCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	response, err := r.client.ResolvePublicUser(callCtx, &authv1.ResolvePublicUserRequest{Username: username})
	if err != nil {
		switch status.Code(err) {
		case codes.InvalidArgument, codes.NotFound:
			return outbound.PublicUser{}, domain.ErrPublicProfileNotFound
		case codes.Canceled:
			if ctx.Err() != nil {
				return outbound.PublicUser{}, ctx.Err()
			}
			return outbound.PublicUser{}, context.Canceled
		case codes.DeadlineExceeded:
			if ctx.Err() != nil {
				return outbound.PublicUser{}, ctx.Err()
			}
			return outbound.PublicUser{}, domain.ErrAuthServiceUnavailable.Wrap(err)
		default:
			return outbound.PublicUser{}, domain.ErrAuthServiceUnavailable.Wrap(err)
		}
	}
	if response == nil || strings.TrimSpace(response.GetUserId()) == "" || strings.TrimSpace(response.GetUsername()) == "" {
		return outbound.PublicUser{}, domain.ErrAuthServiceUnavailable
	}

	return outbound.PublicUser{ID: response.GetUserId(), Username: response.GetUsername()}, nil
}
