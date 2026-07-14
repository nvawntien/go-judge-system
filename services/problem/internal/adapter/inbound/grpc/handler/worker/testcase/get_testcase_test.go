package testcase

import (
	"context"
	"errors"
	"testing"

	problemv1 "go-judge-system/pkg/pb/problem/v1"
	"go-judge-system/services/problem/internal/application/dto"
	"go-judge-system/services/problem/internal/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeGetTestCaseUseCase struct {
	result       *dto.InternalTestCaseResponse
	err          error
	problemID    int64
	executeCalls int
}

func (f *fakeGetTestCaseUseCase) Execute(
	_ context.Context,
	problemID int64,
) (*dto.InternalTestCaseResponse, error) {
	f.problemID = problemID
	f.executeCalls++
	return f.result, f.err
}

func TestGetTestCaseHandlerHandle(t *testing.T) {
	t.Parallel()

	useCase := &fakeGetTestCaseUseCase{
		result: &dto.InternalTestCaseResponse{
			TestCount:      12,
			Version:        3,
			ZipDownloadURL: "https://storage.internal/testcases.zip",
		},
	}
	handler := NewGetTestCaseHandler(useCase)

	response, err := handler.Handle(context.Background(), &problemv1.GetTestCaseRequest{ProblemId: 42})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if useCase.problemID != 42 {
		t.Fatalf("use case problem ID = %d, want 42", useCase.problemID)
	}
	if response.GetZipDownloadUrl() != useCase.result.ZipDownloadURL {
		t.Fatalf("zip download URL = %q, want %q", response.GetZipDownloadUrl(), useCase.result.ZipDownloadURL)
	}
	if response.GetTestCount() != int32(useCase.result.TestCount) {
		t.Fatalf("test count = %d, want %d", response.GetTestCount(), useCase.result.TestCount)
	}
	if response.GetVersion() != int32(useCase.result.Version) {
		t.Fatalf("version = %d, want %d", response.GetVersion(), useCase.result.Version)
	}
}

func TestGetTestCaseHandlerRejectsInvalidProblemID(t *testing.T) {
	t.Parallel()

	for _, request := range []*problemv1.GetTestCaseRequest{
		nil,
		{},
		{ProblemId: -1},
	} {
		useCase := &fakeGetTestCaseUseCase{}
		handler := NewGetTestCaseHandler(useCase)

		_, err := handler.Handle(context.Background(), request)
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("Handle(%v) code = %s, want %s", request, status.Code(err), codes.InvalidArgument)
		}
		if useCase.executeCalls != 0 {
			t.Fatalf("use case execute calls = %d, want 0", useCase.executeCalls)
		}
	}
}

func TestGetTestCaseHandlerMapsUseCaseErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{name: "test case not found", err: domain.ErrTestCaseNotFound, code: codes.NotFound},
		{name: "unexpected error", err: errors.New("storage unavailable"), code: codes.Internal},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			handler := NewGetTestCaseHandler(&fakeGetTestCaseUseCase{err: test.err})
			_, err := handler.Handle(context.Background(), &problemv1.GetTestCaseRequest{ProblemId: 42})
			if status.Code(err) != test.code {
				t.Fatalf("Handle() code = %s, want %s", status.Code(err), test.code)
			}
		})
	}
}
