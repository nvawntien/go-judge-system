package problem

import (
	"context"
	"errors"
	"testing"

	problemv1 "go-judge-system/pkg/pb/problem/v1"
	submissioninbound "go-judge-system/services/problem/internal/application/port/inbound/submission"
	"go-judge-system/services/problem/internal/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeGetProblemForSubmissionUseCase struct {
	result submissioninbound.GetProblemForSubmissionResult
	err    error
	calls  int
}

func (f *fakeGetProblemForSubmissionUseCase) Execute(
	_ context.Context,
	_ int64,
) (submissioninbound.GetProblemForSubmissionResult, error) {
	f.calls++
	return f.result, f.err
}

func TestGetProblemForSubmissionHandlerSuccess(t *testing.T) {
	useCase := &fakeGetProblemForSubmissionUseCase{result: submissioninbound.GetProblemForSubmissionResult{
		ProblemID: 42,
		Title:     "Two Sum",
		Slug:      "two-sum",
	}}

	got, err := NewGetProblemForSubmissionHandler(useCase).Handle(
		context.Background(),
		&problemv1.GetProblemForSubmissionRequest{ProblemId: 42},
	)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if got.GetProblemId() != 42 || got.GetTitle() != "Two Sum" || got.GetSlug() != "two-sum" {
		t.Fatalf("response = %+v", got)
	}
}

func TestGetProblemForSubmissionHandlerRejectsInvalidID(t *testing.T) {
	for _, req := range []*problemv1.GetProblemForSubmissionRequest{nil, {}, {ProblemId: -1}} {
		useCase := &fakeGetProblemForSubmissionUseCase{}
		_, err := NewGetProblemForSubmissionHandler(useCase).Handle(context.Background(), req)
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("Handle(%v) code = %s, want %s", req, status.Code(err), codes.InvalidArgument)
		}
		if useCase.calls != 0 {
			t.Fatalf("use case calls = %d, want 0", useCase.calls)
		}
	}
}

func TestGetProblemForSubmissionHandlerMapsErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{name: "missing", err: domain.ErrProblemNotFound, code: codes.NotFound},
		{name: "canceled", err: context.Canceled, code: codes.Canceled},
		{name: "deadline", err: context.DeadlineExceeded, code: codes.DeadlineExceeded},
		{name: "repository", err: errors.New("database unavailable"), code: codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewGetProblemForSubmissionHandler(&fakeGetProblemForSubmissionUseCase{err: tt.err})
			_, err := handler.Handle(context.Background(), &problemv1.GetProblemForSubmissionRequest{ProblemId: 42})
			if status.Code(err) != tt.code {
				t.Fatalf("Handle() code = %s, want %s", status.Code(err), tt.code)
			}
		})
	}
}
