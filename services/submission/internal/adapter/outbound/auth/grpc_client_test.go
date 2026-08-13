package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	authv1 "go-judge-system/pkg/pb/auth/v1"
	"go-judge-system/services/submission/internal/domain"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakePublicUserServiceClient struct {
	authv1.PublicUserServiceClient
	response *authv1.ResolvePublicUserResponse
	err      error
	req      *authv1.ResolvePublicUserRequest
}

func (f *fakePublicUserServiceClient) ResolvePublicUser(ctx context.Context, req *authv1.ResolvePublicUserRequest, _ ...grpc.CallOption) (*authv1.ResolvePublicUserResponse, error) {
	f.req = req
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	return f.response, f.err
}

func TestGRPCPublicUserResolverMapsContractAndStatuses(t *testing.T) {
	client := &fakePublicUserServiceClient{response: &authv1.ResolvePublicUserResponse{UserId: "user-1", Username: "ada"}}
	got, err := NewGRPCPublicUserResolver(client, time.Second).ResolvePublicUser(context.Background(), "ada")
	if err != nil || got.ID != "user-1" || got.Username != "ada" || client.req.GetUsername() != "ada" {
		t.Fatalf("result/request/error = %+v/%+v/%v", got, client.req, err)
	}

	for _, tt := range []struct {
		code codes.Code
		want error
	}{
		{codes.InvalidArgument, domain.ErrPublicProfileNotFound},
		{codes.NotFound, domain.ErrPublicProfileNotFound},
		{codes.DeadlineExceeded, domain.ErrAuthServiceUnavailable},
		{codes.Unavailable, domain.ErrAuthServiceUnavailable},
		{codes.Internal, domain.ErrAuthServiceUnavailable},
	} {
		_, err := NewGRPCPublicUserResolver(&fakePublicUserServiceClient{err: status.Error(tt.code, "transport")}, time.Second).ResolvePublicUser(context.Background(), "ada")
		if !errors.Is(err, tt.want) {
			t.Fatalf("code %s error = %v, want %v", tt.code, err, tt.want)
		}
	}

	_, err = NewGRPCPublicUserResolver(&fakePublicUserServiceClient{response: &authv1.ResolvePublicUserResponse{}}, time.Second).ResolvePublicUser(context.Background(), "ada")
	if !errors.Is(err, domain.ErrAuthServiceUnavailable) {
		t.Fatalf("malformed response error = %v", err)
	}
}
