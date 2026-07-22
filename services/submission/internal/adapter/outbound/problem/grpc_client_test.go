package problem

import (
	"context"
	"errors"
	"testing"
	"time"

	problemv1 "go-judge-system/pkg/pb/problem/v1"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeProblemServiceClient struct {
	problemv1.ProblemServiceClient
	response *problemv1.GetProblemResponse
	err      error
	request  *problemv1.GetProblemRequest
	call     func(context.Context, *problemv1.GetProblemRequest) (*problemv1.GetProblemResponse, error)
}

func (f *fakeProblemServiceClient) GetProblem(
	ctx context.Context,
	req *problemv1.GetProblemRequest,
	_ ...grpc.CallOption,
) (*problemv1.GetProblemResponse, error) {
	f.request = req
	if f.call != nil {
		return f.call(ctx, req)
	}
	return f.response, f.err
}

func TestGRPCProblemReaderMapsSuccess(t *testing.T) {
	client := &fakeProblemServiceClient{response: &problemv1.GetProblemResponse{
		ProblemId: 42,
		Title:     "Two Sum",
		Slug:      "two-sum",
	}}
	var reader outbound.ProblemReader = NewGRPCProblemReader(client, time.Second)

	got, err := reader.GetProblem(context.Background(), 42, outbound.ProblemActor{UserID: "contributor-a", Role: rbac.RoleContributor})
	if err != nil {
		t.Fatalf("GetProblem() error = %v", err)
	}
	if got.ID != 42 || got.Title != "Two Sum" || got.Slug != "two-sum" {
		t.Fatalf("result = %+v", got)
	}
	if client.request.GetProblemId() != 42 || client.request.GetActorUserId() != "contributor-a" || client.request.GetActorRole() != "contributor" {
		t.Fatalf("request = %+v", client.request)
	}
}

func TestGRPCProblemReaderMapsStatuses(t *testing.T) {
	tests := []struct {
		name string
		code codes.Code
		want error
	}{
		{name: "invalid argument", code: codes.InvalidArgument, want: domain.ErrInvalidProblemID},
		{name: "unauthenticated", code: codes.Unauthenticated, want: domain.ErrProblemActorUnauthenticated},
		{name: "permission denied", code: codes.PermissionDenied, want: domain.ErrProblemActorForbidden},
		{name: "not found", code: codes.NotFound, want: domain.ErrProblemNotFound},
		{name: "deadline", code: codes.DeadlineExceeded, want: domain.ErrProblemServiceUnavailable},
		{name: "unavailable", code: codes.Unavailable, want: domain.ErrProblemServiceUnavailable},
		{name: "internal", code: codes.Internal, want: domain.ErrProblemServiceUnavailable},
		{name: "unknown", code: codes.Unknown, want: domain.ErrProblemServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewGRPCProblemReader(
				&fakeProblemServiceClient{err: status.Error(tt.code, "transport detail")},
				time.Second,
			)
			_, err := reader.GetProblem(context.Background(), 42, outbound.ProblemActor{UserID: "user-1", Role: rbac.RoleUser})
			if !errors.Is(err, tt.want) {
				t.Fatalf("GetProblem() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestGRPCProblemReaderPreservesRequestCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := &fakeProblemServiceClient{call: func(ctx context.Context, _ *problemv1.GetProblemRequest) (*problemv1.GetProblemResponse, error) {
		return nil, status.FromContextError(ctx.Err()).Err()
	}}

	_, err := NewGRPCProblemReader(client, time.Second).GetProblem(ctx, 42, outbound.ProblemActor{UserID: "user-1", Role: rbac.RoleUser})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetProblem() error = %v, want context canceled", err)
	}
}

func TestGRPCProblemReaderRejectsMalformedSuccess(t *testing.T) {
	client := &fakeProblemServiceClient{response: &problemv1.GetProblemResponse{ProblemId: 42}}
	_, err := NewGRPCProblemReader(client, time.Second).GetProblem(context.Background(), 42, outbound.ProblemActor{UserID: "user-1", Role: rbac.RoleUser})
	if !errors.Is(err, domain.ErrProblemServiceUnavailable) {
		t.Fatalf("GetProblem() error = %v, want service unavailable", err)
	}
}

func TestGRPCProblemReaderAppliesConfiguredTimeout(t *testing.T) {
	const timeout = 20 * time.Millisecond
	deadlineSeen := false
	client := &fakeProblemServiceClient{call: func(ctx context.Context, _ *problemv1.GetProblemRequest) (*problemv1.GetProblemResponse, error) {
		_, deadlineSeen = ctx.Deadline()
		<-ctx.Done()
		return nil, status.FromContextError(ctx.Err()).Err()
	}}

	started := time.Now()
	_, err := NewGRPCProblemReader(client, timeout).GetProblem(context.Background(), 42, outbound.ProblemActor{UserID: "user-1", Role: rbac.RoleUser})
	if !errors.Is(err, domain.ErrProblemServiceUnavailable) {
		t.Fatalf("GetProblem() error = %v, want service unavailable", err)
	}
	if !deadlineSeen {
		t.Fatal("gRPC call context had no deadline")
	}
	if elapsed := time.Since(started); elapsed < timeout || elapsed > 500*time.Millisecond {
		t.Fatalf("elapsed = %s, want configured timeout near %s", elapsed, timeout)
	}
}
