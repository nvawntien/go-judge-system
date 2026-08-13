package handler

import (
	"context"
	"errors"
	"testing"

	authv1 "go-judge-system/pkg/pb/auth/v1"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeResolvePublicUserUseCase struct {
	response dto.ResolvePublicUserResponse
	err      error
	req      dto.ResolvePublicUserRequest
}

func (f *fakeResolvePublicUserUseCase) Execute(_ context.Context, req dto.ResolvePublicUserRequest) (dto.ResolvePublicUserResponse, error) {
	f.req = req
	return f.response, f.err
}

func TestResolvePublicUserHandlerMapsContract(t *testing.T) {
	uc := &fakeResolvePublicUserUseCase{response: dto.ResolvePublicUserResponse{UserID: "user-1", Username: "ada"}}
	got, err := NewResolvePublicUserHandler(uc).Handle(context.Background(), &authv1.ResolvePublicUserRequest{Username: "ada"})
	if err != nil || got.GetUserId() != "user-1" || got.GetUsername() != "ada" || uc.req.Username != "ada" {
		t.Fatalf("response/request/error = %+v/%+v/%v", got, uc.req, err)
	}

	for _, tt := range []struct {
		name string
		err  error
		code codes.Code
	}{
		{name: "not found", err: domain.ErrUserNotFound, code: codes.NotFound},
		{name: "canceled", err: context.Canceled, code: codes.Canceled},
		{name: "deadline", err: context.DeadlineExceeded, code: codes.DeadlineExceeded},
		{name: "internal", err: errors.New("db"), code: codes.Internal},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewResolvePublicUserHandler(&fakeResolvePublicUserUseCase{err: tt.err}).Handle(context.Background(), &authv1.ResolvePublicUserRequest{Username: "ada"})
			if status.Code(err) != tt.code {
				t.Fatalf("status = %s, want %s; err=%v", status.Code(err), tt.code, err)
			}
		})
	}

	_, err = NewResolvePublicUserHandler(uc).Handle(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("nil request status = %s", status.Code(err))
	}
}
