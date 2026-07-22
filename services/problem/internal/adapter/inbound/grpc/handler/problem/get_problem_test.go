package problem

import (
	"context"
	"errors"
	"testing"

	problemv1 "go-judge-system/pkg/pb/problem/v1"
	"go-judge-system/pkg/rbac"
	inbound "go-judge-system/services/problem/internal/application/port/inbound"
	"go-judge-system/services/problem/internal/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeGetProblemUseCase struct {
	result inbound.ProblemMetadata
	err    error
	req    inbound.GetProblemRequest
	calls  int
}

func (f *fakeGetProblemUseCase) Execute(
	_ context.Context,
	req inbound.GetProblemRequest,
) (inbound.ProblemMetadata, error) {
	f.req = req
	f.calls++
	return f.result, f.err
}

func TestGetProblemHandlerSuccess(t *testing.T) {
	useCase := &fakeGetProblemUseCase{result: inbound.ProblemMetadata{ID: 42, Title: "Two Sum", Slug: "two-sum"}}
	got, err := NewGetProblemHandler(useCase).Handle(context.Background(), &problemv1.GetProblemRequest{
		ProblemId:   42,
		ActorUserId: "contributor-a",
		ActorRole:   string(rbac.RoleContributor),
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if got.GetProblemId() != 42 || got.GetTitle() != "Two Sum" || got.GetSlug() != "two-sum" {
		t.Fatalf("response = %+v", got)
	}
	if useCase.req.ProblemID != 42 || useCase.req.ActorUserID != "contributor-a" || useCase.req.ActorRole != rbac.RoleContributor {
		t.Fatalf("application request = %+v", useCase.req)
	}
}

func TestGetProblemHandlerRejectsNilRequest(t *testing.T) {
	useCase := &fakeGetProblemUseCase{}
	_, err := NewGetProblemHandler(useCase).Handle(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument || useCase.calls != 0 {
		t.Fatalf("code/calls = %s/%d", status.Code(err), useCase.calls)
	}
}

func TestGetProblemHandlerMapsErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{name: "invalid argument", err: domain.ErrInvalidInput, code: codes.InvalidArgument},
		{name: "missing actor", err: domain.ErrActorUnauthenticated, code: codes.Unauthenticated},
		{name: "unsupported role", err: domain.ErrPermissionDenied, code: codes.PermissionDenied},
		{name: "hidden unauthorized", err: domain.ErrProblemNotFound, code: codes.NotFound},
		{name: "canceled", err: context.Canceled, code: codes.Canceled},
		{name: "deadline", err: context.DeadlineExceeded, code: codes.DeadlineExceeded},
		{name: "repository", err: errors.New("database unavailable"), code: codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewGetProblemHandler(&fakeGetProblemUseCase{err: tt.err})
			_, err := handler.Handle(context.Background(), &problemv1.GetProblemRequest{ProblemId: 42, ActorUserId: "actor", ActorRole: "user"})
			if status.Code(err) != tt.code {
				t.Fatalf("Handle() code = %s, want %s", status.Code(err), tt.code)
			}
		})
	}
}
