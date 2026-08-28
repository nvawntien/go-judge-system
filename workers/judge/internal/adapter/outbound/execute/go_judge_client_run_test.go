package execute

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	judgepb "github.com/criyle/go-judge/pb"
	"go-judge-system/workers/judge/internal/application/port/outbound"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type fakeExecutorRPC struct {
	mu      sync.Mutex
	calls   []*judgepb.Request
	handler func(context.Context, *judgepb.Request) (*judgepb.Response, error)
}

type unaryExecutorServer struct {
	judgepb.UnimplementedExecutorServer
	requests chan *judgepb.Request
}

func (s *unaryExecutorServer) Exec(_ context.Context, req *judgepb.Request) (*judgepb.Response, error) {
	s.requests <- req
	return acceptedResults("ok\n"), nil
}

func (f *fakeExecutorRPC) Exec(ctx context.Context, req *judgepb.Request, _ ...grpc.CallOption) (*judgepb.Response, error) {
	f.mu.Lock()
	f.calls = append(f.calls, req)
	f.mu.Unlock()
	return f.handler(ctx, req)
}

func (f *fakeExecutorRPC) requests() []*judgepb.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*judgepb.Request(nil), f.calls...)
}

func TestGoJudgeClientCompileAndRunRequestsUseProtobufSemantics(t *testing.T) {
	var phase int
	rpc := &fakeExecutorRPC{handler: func(_ context.Context, req *judgepb.Request) (*judgepb.Response, error) {
		phase++
		if phase == 1 {
			assertCompileRequest(t, req)
			return compileAccepted("exe-1"), nil
		}
		assertOfficialRunRequest(t, req, "exe-1", []string{"1\n", "2\n"})
		return acceptedResults("1\n", "2\n"), nil
	}}

	result, err := NewGoJudgeClient(rpc, zap.NewNop()).Execute(context.Background(), outbound.ExecutionRequest{
		Language:           "CPP",
		SourceCode:         "int main() {}",
		StopOnFirstFailure: true,
		Limits: outbound.ExecutionLimits{
			TimeLimitMS:      1_000,
			MemoryLimitKB:    64 * 1024,
			OutputLimitBytes: 4 * 1024,
		},
		TestCases: []outbound.ExecutionTestCase{
			{Index: 1, ID: "1", Kind: "official", Stdin: "1\n", ExpectedOutput: stringPtr("1\n")},
			{Index: 2, ID: "2", Kind: "official", Stdin: "2\n", ExpectedOutput: stringPtr("2\n")},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if phase != 2 {
		t.Fatalf("Exec calls = %d, want compile plus one batch", phase)
	}
	if result.Status != "ACCEPTED" || len(result.TestCases) != 2 {
		t.Fatalf("result = %#v, want two accepted testcases", result)
	}
}

func TestGoJudgeClientUsesOfficialUnaryGRPCExecutor(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	executor := &unaryExecutorServer{requests: make(chan *judgepb.Request, 1)}
	judgepb.RegisterExecutorServer(server, executor)
	go func() { _ = server.Serve(listener) }()
	defer server.Stop()

	conn, err := grpc.NewClient(
		"passthrough:///go-judge-test",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("create bufconn client: %v", err)
	}
	defer func() { _ = conn.Close() }()

	result, err := NewGoJudgeClient(judgepb.NewExecutorClient(conn), zap.NewNop()).Execute(context.Background(), outbound.ExecutionRequest{
		Language: "PYTHON", SourceCode: "print('ok')", TestCases: []outbound.ExecutionTestCase{{Index: 1, ID: "case", Stdin: "input\n"}},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != "ACCEPTED" || len(result.TestCases) != 1 {
		t.Fatalf("result = %#v", result)
	}
	select {
	case req := <-executor.requests:
		if got := string(req.GetCmd()[0].GetCopyIn()["main.py"].GetMemory().GetContent()); got != "print('ok')" {
			t.Fatalf("protobuf source = %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("official gRPC Executor.Exec did not receive a request")
	}
}

func TestGoJudgeClientCompileErrorPreservesDiagnostics(t *testing.T) {
	rpc := &fakeExecutorRPC{handler: func(_ context.Context, _ *judgepb.Request) (*judgepb.Response, error) {
		return &judgepb.Response{Results: []*judgepb.Response_Result{{
			Status: judgepb.Response_Result_NonZeroExitStatus,
			Files:  map[string][]byte{"stderr": []byte("./main.go:19:9: make (built-in) must be called")},
		}}}, nil
	}}

	result, err := NewGoJudgeClient(rpc, zap.NewNop()).Execute(context.Background(), outbound.ExecutionRequest{
		Language:   "GO",
		SourceCode: "bad",
		TestCases:  []outbound.ExecutionTestCase{{Index: 1, ID: "sample", Kind: "sample"}},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != "COMPILATION_ERROR" || result.CompileOutput == nil || *result.CompileOutput != "main.go:19:9: make (built-in) must be called" {
		t.Fatalf("compile result = %#v", result)
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Line != 19 || result.Diagnostics[0].Column != 9 {
		t.Fatalf("compile diagnostics = %#v", result.Diagnostics)
	}
}

func TestGoJudgeClientOfficialBatchingAndEarlyTermination(t *testing.T) {
	t.Run("24 accepted testcases use six batches of four", func(t *testing.T) {
		var batchSizes []int
		rpc := &fakeExecutorRPC{handler: func(_ context.Context, req *judgepb.Request) (*judgepb.Response, error) {
			if isCompileRequest(req) {
				return compileAccepted("exe-1"), nil
			}
			batchSizes = append(batchSizes, len(req.GetCmd()))
			return acceptedForRequest(req), nil
		}}
		result, err := NewGoJudgeClient(rpc, zap.NewNop()).Execute(context.Background(), outbound.ExecutionRequest{
			Language: "CPP", SourceCode: "int main(){}", StopOnFirstFailure: true, TestCases: officialTestCases(24),
		})
		if err != nil {
			t.Fatal(err)
		}
		if got, want := fmt.Sprint(batchSizes), "[4 4 4 4 4 4]"; got != want {
			t.Fatalf("batch sizes = %s, want %s", got, want)
		}
		if result.Status != "ACCEPTED" || len(result.TestCases) != 24 {
			t.Fatalf("result = %#v", result)
		}
	})

	t.Run("failure prevents later batches", func(t *testing.T) {
		var batchCount int
		rpc := &fakeExecutorRPC{handler: func(_ context.Context, req *judgepb.Request) (*judgepb.Response, error) {
			if isCompileRequest(req) {
				return compileAccepted("exe-1"), nil
			}
			batchCount++
			response := acceptedForRequest(req)
			if batchCount == 2 {
				response.Results[1].Files["stdout"] = []byte("wrong\n")
			}
			return response, nil
		}}
		result, err := NewGoJudgeClient(rpc, zap.NewNop()).Execute(context.Background(), outbound.ExecutionRequest{
			Language: "CPP", SourceCode: "int main(){}", StopOnFirstFailure: true, TestCases: officialTestCases(24),
		})
		if err != nil {
			t.Fatal(err)
		}
		if batchCount != 2 || result.Status != "WRONG_ANSWER" || len(result.TestCases) != 6 {
			t.Fatalf("batches/result = %d/%#v, want 2 batches and ordered testcase 6 failure", batchCount, result)
		}
	})
}

func TestGoJudgeClientNonEarlyStopPreservesFiftyCommandBatch(t *testing.T) {
	for _, tt := range []struct {
		count int
		want  string
	}{{24, "[24]"}, {50, "[50]"}, {51, "[50 1]"}} {
		t.Run(fmt.Sprintf("%d testcases", tt.count), func(t *testing.T) {
			var batchSizes []int
			rpc := &fakeExecutorRPC{handler: func(_ context.Context, req *judgepb.Request) (*judgepb.Response, error) {
				if isCompileRequest(req) {
					return compileAccepted("exe-1"), nil
				}
				batchSizes = append(batchSizes, len(req.GetCmd()))
				return acceptedForRequest(req), nil
			}}
			result, err := NewGoJudgeClient(rpc, zap.NewNop()).Execute(context.Background(), outbound.ExecutionRequest{
				Language: "CPP", SourceCode: "int main(){}", TestCases: officialTestCases(tt.count),
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := fmt.Sprint(batchSizes); got != tt.want {
				t.Fatalf("batch sizes = %s, want %s", got, tt.want)
			}
			if len(result.TestCases) != tt.count {
				t.Fatalf("testcase count = %d, want %d", len(result.TestCases), tt.count)
			}
		})
	}
}

func TestGoJudgeClientMapsExecutionResponses(t *testing.T) {
	testCase := outbound.ExecutionTestCase{Index: 1, ID: "case", Kind: "custom"}
	for _, tt := range []struct {
		name       string
		status     judgepb.Response_Result_StatusType
		exitStatus int32
		want       string
	}{
		{"accepted", judgepb.Response_Result_Accepted, 0, "ACCEPTED"},
		{"accepted nonzero exit", judgepb.Response_Result_Accepted, 1, "RUNTIME_ERROR"},
		{"tle", judgepb.Response_Result_TimeLimitExceeded, 0, "TIME_LIMIT_EXCEEDED"},
		{"mle", judgepb.Response_Result_MemoryLimitExceeded, 0, "MEMORY_LIMIT_EXCEEDED"},
		{"output limit", judgepb.Response_Result_OutputLimitExceeded, 0, "OUTPUT_LIMIT_EXCEEDED"},
		{"runtime", judgepb.Response_Result_NonZeroExitStatus, 1, "RUNTIME_ERROR"},
		{"internal", judgepb.Response_Result_InternalError, 0, "SYSTEM_ERROR"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := mapTestCaseResult("CPP", testCase, &judgepb.Response_Result{Status: tt.status, ExitStatus: tt.exitStatus}, false)
			if got.Status != tt.want {
				t.Fatalf("status = %s, want %s", got.Status, tt.want)
			}
		})
	}

	accepted := mapTestCaseResult("CPP", outbound.ExecutionTestCase{Index: 1, ID: "wa", ExpectedOutput: stringPtr("expected\n")}, &judgepb.Response_Result{
		Status: judgepb.Response_Result_Accepted,
		Files:  map[string][]byte{"stdout": []byte("actual\n")},
	}, true)
	if accepted.Status != "WRONG_ANSWER" {
		t.Fatalf("output comparison status = %s, want WRONG_ANSWER", accepted.Status)
	}
}

func TestGoJudgeClientPropagatesCancellationToUnaryRPC(t *testing.T) {
	called := make(chan struct{})
	rpc := &fakeExecutorRPC{handler: func(ctx context.Context, _ *judgepb.Request) (*judgepb.Response, error) {
		close(called)
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewGoJudgeClient(rpc, zap.NewNop()).Execute(ctx, outbound.ExecutionRequest{
		Language: "GO", SourceCode: "package main", TestCases: []outbound.ExecutionTestCase{{Index: 1, ID: "case"}},
	})
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("Execute() error = %v, want wrapped context cancellation", err)
	}
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("cancelled context was not passed to unary RPC")
	}
}

func TestGoJudgeClientRejectsMalformedResponses(t *testing.T) {
	for _, tt := range []struct {
		name string
		resp *judgepb.Response
		want string
	}{
		{"missing executable file ID", compileAccepted(""), "executable file ID"},
		{"nil compile result", &judgepb.Response{Results: []*judgepb.Response_Result{nil}}, "did not contain a result"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rpc := &fakeExecutorRPC{handler: func(_ context.Context, _ *judgepb.Request) (*judgepb.Response, error) { return tt.resp, nil }}
			_, err := NewGoJudgeClient(rpc, zap.NewNop()).Execute(context.Background(), outbound.ExecutionRequest{
				Language: "CPP", SourceCode: "int main(){}", TestCases: []outbound.ExecutionTestCase{{Index: 1, ID: "case"}},
			})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}

	t.Run("run result count mismatch", func(t *testing.T) {
		var calls int
		rpc := &fakeExecutorRPC{handler: func(_ context.Context, _ *judgepb.Request) (*judgepb.Response, error) {
			calls++
			if calls == 1 {
				return compileAccepted("exe-1"), nil
			}
			return &judgepb.Response{Results: []*judgepb.Response_Result{}}, nil
		}}
		_, err := NewGoJudgeClient(rpc, zap.NewNop()).Execute(context.Background(), outbound.ExecutionRequest{
			Language: "CPP", SourceCode: "int main(){}", TestCases: []outbound.ExecutionTestCase{{Index: 1, ID: "case"}},
		})
		if err == nil || !strings.Contains(err.Error(), "returned 0 results") {
			t.Fatalf("error = %v, want result count mismatch", err)
		}
	})
}

func TestGoJudgeClientRaisesOutputLimitForLargeExpectedOutput(t *testing.T) {
	expected := strings.Repeat("7 ", 600*1024)
	var compile bool
	rpc := &fakeExecutorRPC{handler: func(_ context.Context, req *judgepb.Request) (*judgepb.Response, error) {
		if isCompileRequest(req) {
			compile = true
			return compileAccepted("exe-1"), nil
		}
		if req.GetCmd()[0].GetFiles()[1].GetPipe().GetMax() < int64(len(expected)+expectedOutputHeadroomBytes) {
			t.Fatalf("stdout collector max = %d, want expected-output headroom", req.GetCmd()[0].GetFiles()[1].GetPipe().GetMax())
		}
		return acceptedResults(expected), nil
	}}
	result, err := NewGoJudgeClient(rpc, zap.NewNop()).Execute(context.Background(), outbound.ExecutionRequest{
		Language: "CPP", SourceCode: "int main(){}", StopOnFirstFailure: true,
		TestCases: []outbound.ExecutionTestCase{{Index: 1, ID: "large", ExpectedOutput: &expected}},
	})
	if err != nil || !compile || result.Status != "ACCEPTED" {
		t.Fatalf("result/error/compile = %#v/%v/%t", result, err, compile)
	}
}

func assertCompileRequest(t *testing.T, req *judgepb.Request) {
	t.Helper()
	if len(req.GetCmd()) != 1 {
		t.Fatalf("compile command count = %d, want 1", len(req.GetCmd()))
	}
	cmd := req.GetCmd()[0]
	if got, want := fmt.Sprint(cmd.GetArgs()), "[/usr/bin/g++ -O3 -std=c++17 main.cpp -o main]"; got != want {
		t.Fatalf("compile args = %s, want %s", got, want)
	}
	if got, want := fmt.Sprint(cmd.GetEnv()), "[PATH=/usr/bin:/bin]"; got != want {
		t.Fatalf("compile env = %s, want %s", got, want)
	}
	if got := string(cmd.GetCopyIn()["main.cpp"].GetMemory().GetContent()); got != "int main() {}" {
		t.Fatalf("compile source = %q", got)
	}
	if cmd.GetFiles()[1].GetPipe().GetName() != "stdout" || cmd.GetFiles()[2].GetPipe().GetName() != "stderr" {
		t.Fatalf("compile collectors = %#v", cmd.GetFiles())
	}
	if got, want := cmd.GetCopyOutCached()[0].GetName(), "main"; got != want {
		t.Fatalf("copy-out cached = %q, want %q", got, want)
	}
	if got, want := cmd.GetCpuTimeLimit(), uint64(30*time.Second); got != want {
		t.Fatalf("compile cpu limit = %v, want %v", time.Duration(got), time.Duration(want))
	}
	if got, want := cmd.GetClockTimeLimit(), uint64(60*time.Second); got != want {
		t.Fatalf("compile clock limit = %v, want %v", time.Duration(got), time.Duration(want))
	}
	if got, want := cmd.GetMemoryLimit(), uint64(512*1024*1024); got != want {
		t.Fatalf("compile memory limit = %d, want %d", got, want)
	}
	if got, want := cmd.GetProcLimit(), uint64(500); got != want {
		t.Fatalf("compile proc limit = %d, want %d", got, want)
	}
}

func assertOfficialRunRequest(t *testing.T, req *judgepb.Request, fileID string, stdin []string) {
	t.Helper()
	if len(req.GetCmd()) != len(stdin) {
		t.Fatalf("run command count = %d, want %d", len(req.GetCmd()), len(stdin))
	}
	for index, cmd := range req.GetCmd() {
		if got := string(cmd.GetFiles()[0].GetMemory().GetContent()); got != stdin[index] {
			t.Fatalf("stdin[%d] = %q, want %q", index, got, stdin[index])
		}
		if got := cmd.GetCopyIn()["main"].GetCached().GetFileID(); got != fileID {
			t.Fatalf("cached executable file ID = %q, want %q", got, fileID)
		}
		if cmd.GetFiles()[1].GetPipe().GetName() != "stdout" || cmd.GetFiles()[2].GetPipe().GetName() != "stderr" {
			t.Fatalf("run collectors = %#v", cmd.GetFiles())
		}
		if got, want := cmd.GetCpuTimeLimit(), uint64(time.Second); got != want {
			t.Fatalf("run cpu limit = %v, want %v", time.Duration(got), time.Duration(want))
		}
		if got, want := cmd.GetClockTimeLimit(), uint64(2*time.Second); got != want {
			t.Fatalf("run clock limit = %v, want %v", time.Duration(got), time.Duration(want))
		}
		if got, want := cmd.GetMemoryLimit(), uint64(64*1024*1024); got != want {
			t.Fatalf("run memory limit = %d, want %d", got, want)
		}
		if got, want := cmd.GetProcLimit(), uint64(50); got != want {
			t.Fatalf("run proc limit = %d, want %d", got, want)
		}
	}
}

func isCompileRequest(req *judgepb.Request) bool {
	return len(req.GetCmd()) == 1 && len(req.GetCmd()[0].GetCopyOutCached()) == 1
}

func compileAccepted(fileID string) *judgepb.Response {
	return &judgepb.Response{Results: []*judgepb.Response_Result{{
		Status:  judgepb.Response_Result_Accepted,
		FileIDs: map[string]string{"main": fileID},
	}}}
}

func acceptedResults(outputs ...string) *judgepb.Response {
	response := &judgepb.Response{Results: make([]*judgepb.Response_Result, 0, len(outputs))}
	for _, output := range outputs {
		response.Results = append(response.Results, &judgepb.Response_Result{
			Status: judgepb.Response_Result_Accepted,
			Files:  map[string][]byte{"stdout": []byte(output), "stderr": []byte{}},
		})
	}
	return response
}

func acceptedForRequest(req *judgepb.Request) *judgepb.Response {
	outputs := make([]string, 0, len(req.GetCmd()))
	for _, cmd := range req.GetCmd() {
		outputs = append(outputs, string(cmd.GetFiles()[0].GetMemory().GetContent()))
	}
	return acceptedResults(outputs...)
}

func officialTestCases(count int) []outbound.ExecutionTestCase {
	testCases := make([]outbound.ExecutionTestCase, 0, count)
	for index := 1; index <= count; index++ {
		output := fmt.Sprintf("%d\n", index)
		testCases = append(testCases, outbound.ExecutionTestCase{
			Index: index, ID: fmt.Sprintf("%d", index), Kind: "official", Stdin: output, ExpectedOutput: &output,
		})
	}
	return testCases
}

func stringPtr(value string) *string { return &value }

var _ executorRPC = (*fakeExecutorRPC)(nil)
