package execute

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-judge-system/pkg/gojudge"
	"go-judge-system/workers/judge/internal/application/port/outbound"

	"go.uber.org/zap"
)

func TestRunCodeCompilesOnceAndRunsAllCases(t *testing.T) {
	var compileCalls int
	var runCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req gojudge.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(req.Cmd) == 1 && req.Cmd[0].CopyOutCached != nil {
			compileCalls++
			_ = json.NewEncoder(w).Encode(gojudge.Response{{Status: "Accepted", FileIDs: map[string]string{"main": "exe-1"}}})
			return
		}

		runCalls++
		if len(req.Cmd) != 3 {
			t.Fatalf("run command count = %d, want 3", len(req.Cmd))
		}
		_ = json.NewEncoder(w).Encode(gojudge.Response{
			{Status: "Accepted", Files: map[string]string{"stdout": "1\n", "stderr": ""}, Time: 1_000_000, Memory: 1024},
			{Status: "Runtime Error", ExitStatus: 1, Files: map[string]string{"stdout": "", "stderr": "panic: boom\nmain.main()\n\t/tmp/run/main.go:9 +0x1"}, Time: 2_000_000, Memory: 2048},
			{Status: "Accepted", Files: map[string]string{"stdout": "3\n", "stderr": ""}, Time: 3_000_000, Memory: 3072},
		})
	}))
	defer server.Close()

	client := NewGoJudgeClient(server.URL, zap.NewNop())
	res, err := client.Execute(context.Background(), outbound.ExecutionRequest{
		Language:           "GO",
		SourceCode:         "package main\nfunc main() {}\n",
		StopOnFirstFailure: false,
		TestCases: []outbound.ExecutionTestCase{
			{Index: 1, ID: "sample-1", Kind: "sample", Stdin: "1\n", ExpectedOutput: stringPtr("1\n")},
			{Index: 2, ID: "sample-2", Kind: "sample", Stdin: "2\n", ExpectedOutput: stringPtr("2\n")},
			{Index: 3, ID: "custom-1", Kind: "custom", Stdin: "3\n"},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if compileCalls != 1 || runCalls != 1 {
		t.Fatalf("compile/run calls = %d/%d, want 1/1", compileCalls, runCalls)
	}
	if got := []string{res.TestCases[0].Status, res.TestCases[1].Status, res.TestCases[2].Status}; got[0] != "ACCEPTED" || got[1] != "RUNTIME_ERROR" || got[2] != "ACCEPTED" {
		t.Fatalf("statuses = %v, want [ACCEPTED RUNTIME_ERROR ACCEPTED]", got)
	}
	if len(res.TestCases[1].Diagnostics) != 1 || res.TestCases[1].Diagnostics[0].Line != 9 {
		t.Fatalf("runtime diagnostics = %+v, want line 9", res.TestCases[1].Diagnostics)
	}
	if res.ErrorMessage == nil || *res.ErrorMessage != "panic: boom" {
		t.Fatalf("error message = %v, want runtime panic headline", res.ErrorMessage)
	}
}

func TestOfficialExecutionRunsOneTestCasePerRequest(t *testing.T) {
	var runCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req gojudge.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(req.Cmd) == 1 && req.Cmd[0].CopyOutCached != nil {
			_ = json.NewEncoder(w).Encode(gojudge.Response{{Status: "Accepted", FileIDs: map[string]string{"main": "exe-1"}}})
			return
		}

		runCalls++
		if len(req.Cmd) != 1 {
			t.Fatalf("official request command count = %d, want 1", len(req.Cmd))
		}
		_ = json.NewEncoder(w).Encode(gojudge.Response{{Status: "Accepted", Files: map[string]string{"stdout": "1\n", "stderr": ""}}})
	}))
	defer server.Close()

	client := NewGoJudgeClient(server.URL, zap.NewNop())
	res, err := client.Execute(context.Background(), outbound.ExecutionRequest{
		Language:           "CPP",
		SourceCode:         "int main(){}",
		StopOnFirstFailure: true,
		TestCases: []outbound.ExecutionTestCase{
			{Index: 1, ID: "1", Kind: "official", Stdin: "1\n", ExpectedOutput: stringPtr("1\n")},
			{Index: 2, ID: "2", Kind: "official", Stdin: "2\n", ExpectedOutput: stringPtr("1\n")},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if runCalls != 2 {
		t.Fatalf("run calls = %d, want 2", runCalls)
	}
	if res.Status != "ACCEPTED" || len(res.TestCases) != 2 {
		t.Fatalf("result = %#v, want two accepted test cases", res)
	}
}

func TestRunCodeCompileErrorReturnsNoTests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(gojudge.Response{{Status: "Nonzero Exit Status", Files: map[string]string{"stderr": "./main.go:19:9: make (built-in) must be called"}}})
	}))
	defer server.Close()

	client := NewGoJudgeClient(server.URL, zap.NewNop())
	res, err := client.Execute(context.Background(), outbound.ExecutionRequest{
		Language:   "GO",
		SourceCode: "bad",
		TestCases:  []outbound.ExecutionTestCase{{Index: 1, ID: "sample-1", Kind: "sample", ExpectedOutput: stringPtr("")}},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if res.Status != "COMPILATION_ERROR" || len(res.TestCases) != 0 {
		t.Fatalf("result = %#v, want COMPILATION_ERROR with no tests", res)
	}
	if res.CompileOutput == nil || *res.CompileOutput != "main.go:19:9: make (built-in) must be called" {
		t.Fatalf("compile output = %v, want sanitized compiler output", res.CompileOutput)
	}
	if len(res.Diagnostics) != 1 || res.Diagnostics[0].Line != 19 || res.Diagnostics[0].Column != 9 {
		t.Fatalf("compile diagnostics = %+v, want line 19 column 9", res.Diagnostics)
	}
	if res.ErrorMessage != nil {
		t.Fatalf("compile error message = %q, want nil", *res.ErrorMessage)
	}
}

func TestMapJudgeStatusClassifiesUserCodeFailures(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		exitStatus int
		want       string
	}{
		{name: "runtime error", status: "Runtime Error", want: "RUNTIME_ERROR"},
		{name: "nonzero exit", status: "Nonzero Exit Status", want: "RUNTIME_ERROR"},
		{name: "tle", status: "Time Limit Exceeded", want: "TIME_LIMIT_EXCEEDED"},
		{name: "mle", status: "Memory Limit Exceeded", want: "MEMORY_LIMIT_EXCEEDED"},
		{name: "ole", status: "Output Limit Exceeded", want: "OUTPUT_LIMIT_EXCEEDED"},
		{name: "internal", status: "Internal Error", want: "SYSTEM_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapJudgeStatus(tt.status, tt.exitStatus); got != tt.want {
				t.Fatalf("mapJudgeStatus(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestExecutePropagatesContextCancellationToGoJudge(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer func() {
		close(release)
		server.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	client := NewGoJudgeClient(server.URL, zap.NewNop())
	_, err := client.Execute(ctx, outbound.ExecutionRequest{
		Language:   "GO",
		SourceCode: "package main\nfunc main() {}\n",
		TestCases:  []outbound.ExecutionTestCase{{Index: 1, ID: "case-1", Kind: "custom", Stdin: "1\n"}},
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want context cancellation error")
	}
	if ctx.Err() == nil {
		t.Fatal("context was not cancelled")
	}
}

func TestSubmissionOutputLimitIsPreserved(t *testing.T) {
	if got := mapOfficialSubmissionStatus(mapJudgeStatus("Output Limit Exceeded", 0)); got != "OUTPUT_LIMIT_EXCEEDED" {
		t.Fatalf("submission status = %q, want OUTPUT_LIMIT_EXCEEDED", got)
	}
	if got := mapJudgeStatus("Output Limit Exceeded", 0); got != "OUTPUT_LIMIT_EXCEEDED" {
		t.Fatalf("raw judge status = %q, want OUTPUT_LIMIT_EXCEEDED for run-code mapper", got)
	}
}

func TestExecuteRaisesOutputLimitForLargeExpectedOutput(t *testing.T) {
	expected := strings.Repeat("7 ", 600*1024)
	var sawRun bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req gojudge.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(req.Cmd) == 1 && req.Cmd[0].CopyOutCached != nil {
			_ = json.NewEncoder(w).Encode(gojudge.Response{{Status: "Accepted", FileIDs: map[string]string{"main": "exe-1"}}})
			return
		}

		sawRun = true
		if len(req.Cmd) != 1 {
			t.Fatalf("run command count = %d, want 1", len(req.Cmd))
		}
		stdoutFile := req.Cmd[0].Files[1]
		if stdoutFile.Max == nil {
			t.Fatal("stdout max = nil")
		}
		wantMin := int64(len(expected) + expectedOutputHeadroomBytes)
		if *stdoutFile.Max < wantMin {
			t.Fatalf("stdout max = %d, want at least %d", *stdoutFile.Max, wantMin)
		}
		_ = json.NewEncoder(w).Encode(gojudge.Response{{
			Status: "Accepted",
			Files:  map[string]string{"stdout": expected, "stderr": ""},
		}})
	}))
	defer server.Close()

	client := NewGoJudgeClient(server.URL, zap.NewNop())
	res, err := client.Execute(context.Background(), outbound.ExecutionRequest{
		Language:           "CPP",
		SourceCode:         "int main(){}",
		StopOnFirstFailure: true,
		TestCases: []outbound.ExecutionTestCase{{
			Index:          1,
			ID:             "large-output",
			Kind:           "official",
			Stdin:          "1\n",
			ExpectedOutput: &expected,
		}},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !sawRun {
		t.Fatal("server did not receive run request")
	}
	if res.Status != "ACCEPTED" || len(res.TestCases) != 1 || res.TestCases[0].Status != "ACCEPTED" {
		t.Fatalf("result = %#v, want accepted large output", res)
	}
}
