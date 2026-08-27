package execute

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestOfficialExecutionBatchesAtMostFourAndCompilesOnce(t *testing.T) {
	tests := []struct {
		name        string
		testCount   int
		wantBatches []int
	}{
		{name: "one testcase", testCount: 1, wantBatches: []int{1}},
		{name: "four testcases", testCount: 4, wantBatches: []int{4}},
		{name: "five testcases", testCount: 5, wantBatches: []int{4, 1}},
		{name: "twenty four testcases", testCount: 24, wantBatches: []int{4, 4, 4, 4, 4, 4}},
		{name: "fifty testcases", testCount: 50, wantBatches: []int{4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var compileCalls int
			var batches []int
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

				batches = append(batches, len(req.Cmd))
				response := make(gojudge.Response, 0, len(req.Cmd))
				for _, cmd := range req.Cmd {
					file := cmd.CopyIn["main"]
					if file == nil || file.FileID == nil || *file.FileID != "exe-1" {
						t.Fatalf("compiled command did not reuse executable file ID: %#v", file)
					}
					response = append(response, acceptedResponseForCommand(t, cmd))
				}
				_ = json.NewEncoder(w).Encode(response)
			}))

			client := NewGoJudgeClient(server.URL, zap.NewNop())
			result, err := client.Execute(context.Background(), outbound.ExecutionRequest{
				Language:           "CPP",
				SourceCode:         "int main(){}",
				StopOnFirstFailure: true,
				TestCases:          officialTestCases(tt.testCount),
			})
			server.Close()
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if compileCalls != 1 {
				t.Fatalf("compile calls = %d, want 1", compileCalls)
			}
			if fmt.Sprint(batches) != fmt.Sprint(tt.wantBatches) {
				t.Fatalf("batch sizes = %v, want %v", batches, tt.wantBatches)
			}
			if result.Status != "ACCEPTED" || len(result.TestCases) != tt.testCount {
				t.Fatalf("result = %#v, want %d accepted test cases", result, tt.testCount)
			}
		})
	}
}

func TestOfficialExecutionStopsAfterFailingBatchInTestcaseOrder(t *testing.T) {
	var batches [][]int
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

		indexes := commandIndexes(t, req.Cmd)
		batches = append(batches, indexes)
		response := make(gojudge.Response, 0, len(indexes))
		for _, index := range indexes {
			if index == 6 {
				response = append(response, gojudge.Result{Status: "Accepted", Files: map[string]string{"stdout": "wrong\n", "stderr": ""}})
				continue
			}
			if index == 7 {
				response = append(response, gojudge.Result{Status: "Time Limit Exceeded", Files: map[string]string{"stdout": "", "stderr": ""}})
				continue
			}
			response = append(response, acceptedResponse(index))
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	result, err := NewGoJudgeClient(server.URL, zap.NewNop()).Execute(context.Background(), outbound.ExecutionRequest{
		Language:           "CPP",
		SourceCode:         "int main(){}",
		StopOnFirstFailure: true,
		TestCases:          officialTestCases(24),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if fmt.Sprint(batches) != "[[1 2 3 4] [5 6 7 8]]" {
		t.Fatalf("submitted batches = %v, want [[1 2 3 4] [5 6 7 8]]", batches)
	}
	if result.Status != "WRONG_ANSWER" || len(result.TestCases) != 6 || result.TestCases[5].ID != "6" {
		t.Fatalf("result = %#v, want first ordered failure at testcase 6", result)
	}
}

func TestOfficialExecutionDoesNotSubmitBatchesAfterLaterFailure(t *testing.T) {
	var batches [][]int
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

		indexes := commandIndexes(t, req.Cmd)
		batches = append(batches, indexes)
		response := make(gojudge.Response, 0, len(indexes))
		for _, index := range indexes {
			if index == 10 {
				response = append(response, gojudge.Result{Status: "Accepted", Files: map[string]string{"stdout": "wrong\n", "stderr": ""}})
				continue
			}
			response = append(response, acceptedResponse(index))
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	result, err := NewGoJudgeClient(server.URL, zap.NewNop()).Execute(context.Background(), outbound.ExecutionRequest{
		Language:           "CPP",
		SourceCode:         "int main(){}",
		StopOnFirstFailure: true,
		TestCases:          officialTestCases(24),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if fmt.Sprint(batches) != "[[1 2 3 4] [5 6 7 8] [9 10 11 12]]" {
		t.Fatalf("submitted batches = %v, want batches through testcase 12 only", batches)
	}
	if result.Status != "WRONG_ANSWER" || len(result.TestCases) != 10 || result.TestCases[9].ID != "10" {
		t.Fatalf("result = %#v, want first ordered failure at testcase 10", result)
	}
}

func TestOfficialExecutionDoesNotSubmitLaterBatchesAfterFirstBatchFailure(t *testing.T) {
	var batches [][]int
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

		indexes := commandIndexes(t, req.Cmd)
		batches = append(batches, indexes)
		response := make(gojudge.Response, 0, len(indexes))
		for _, index := range indexes {
			if index == 2 {
				response = append(response, gojudge.Result{Status: "Accepted", Files: map[string]string{"stdout": "wrong\n", "stderr": ""}})
				continue
			}
			response = append(response, acceptedResponse(index))
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	result, err := NewGoJudgeClient(server.URL, zap.NewNop()).Execute(context.Background(), outbound.ExecutionRequest{
		Language:           "CPP",
		SourceCode:         "int main(){}",
		StopOnFirstFailure: true,
		TestCases:          officialTestCases(24),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if fmt.Sprint(batches) != "[[1 2 3 4]]" {
		t.Fatalf("submitted batches = %v, want only the first batch", batches)
	}
	if result.Status != "WRONG_ANSWER" || len(result.TestCases) != 2 || result.TestCases[1].ID != "2" {
		t.Fatalf("result = %#v, want first ordered failure at testcase 2", result)
	}
}

func TestExecutionWithoutEarlyStopProcessesAllBatches(t *testing.T) {
	tests := []struct {
		name        string
		testCount   int
		wantBatches []int
	}{
		{name: "twenty four testcases", testCount: 24, wantBatches: []int{24}},
		{name: "fifty testcases", testCount: 50, wantBatches: []int{50}},
		{name: "fifty one testcases", testCount: 51, wantBatches: []int{50, 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var compileCalls int
			var batches []int
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

				batches = append(batches, len(req.Cmd))
				response := make(gojudge.Response, 0, len(req.Cmd))
				for _, cmd := range req.Cmd {
					response = append(response, acceptedResponseForCommand(t, cmd))
				}
				_ = json.NewEncoder(w).Encode(response)
			}))

			result, err := NewGoJudgeClient(server.URL, zap.NewNop()).Execute(context.Background(), outbound.ExecutionRequest{
				Language:           "CPP",
				SourceCode:         "int main(){}",
				StopOnFirstFailure: false,
				TestCases:          officialTestCases(tt.testCount),
			})
			server.Close()
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if compileCalls != 1 {
				t.Fatalf("compile calls = %d, want 1", compileCalls)
			}
			if fmt.Sprint(batches) != fmt.Sprint(tt.wantBatches) {
				t.Fatalf("batch sizes = %v, want %v", batches, tt.wantBatches)
			}
			if result.Status != "ACCEPTED" || len(result.TestCases) != tt.testCount {
				t.Fatalf("result = %#v, want %d accepted test cases", result, tt.testCount)
			}
		})
	}
}

func officialTestCases(count int) []outbound.ExecutionTestCase {
	testCases := make([]outbound.ExecutionTestCase, 0, count)
	for index := 1; index <= count; index++ {
		output := fmt.Sprintf("%d\n", index)
		testCases = append(testCases, outbound.ExecutionTestCase{
			Index:          index,
			ID:             fmt.Sprintf("%d", index),
			Kind:           "official",
			Stdin:          output,
			ExpectedOutput: &output,
		})
	}
	return testCases
}

func commandIndexes(t *testing.T, commands []*gojudge.Cmd) []int {
	t.Helper()
	indexes := make([]int, 0, len(commands))
	for _, command := range commands {
		if len(command.Files) == 0 || command.Files[0].Content == nil {
			t.Fatalf("command stdin = %#v, want a testcase index", command.Files)
		}
		var index int
		if _, err := fmt.Sscanf(*command.Files[0].Content, "%d", &index); err != nil {
			t.Fatalf("parse testcase stdin %q: %v", *command.Files[0].Content, err)
		}
		indexes = append(indexes, index)
	}
	return indexes
}

func acceptedResponseForCommand(t *testing.T, command *gojudge.Cmd) gojudge.Result {
	t.Helper()
	indexes := commandIndexes(t, []*gojudge.Cmd{command})
	return acceptedResponse(indexes[0])
}

func acceptedResponse(index int) gojudge.Result {
	return gojudge.Result{Status: "Accepted", Files: map[string]string{"stdout": fmt.Sprintf("%d\n", index), "stderr": ""}}
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

func TestCompileUsesMinimumCompileBudget(t *testing.T) {
	var sawCompile bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req gojudge.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		if len(req.Cmd) == 1 && req.Cmd[0].CopyOutCached != nil {
			sawCompile = true

			cmd := req.Cmd[0]

			wantCPU := uint64(30 * time.Second)
			wantClock := uint64(60 * time.Second)

			if cmd.CPULimit != wantCPU {
				t.Fatalf(
					"compile CPU limit = %v, want %v",
					time.Duration(cmd.CPULimit),
					time.Duration(wantCPU),
				)
			}

			if cmd.ClockLimit != wantClock {
				t.Fatalf(
					"compile clock limit = %v, want %v",
					time.Duration(cmd.ClockLimit),
					time.Duration(wantClock),
				)
			}

			_ = json.NewEncoder(w).Encode(gojudge.Response{{
				Status:  "Accepted",
				FileIDs: map[string]string{"main": "exe-1"},
			}})
			return
		}

		_ = json.NewEncoder(w).Encode(gojudge.Response{{
			Status: "Accepted",
			Files: map[string]string{
				"stdout": "42\n",
				"stderr": "",
			},
		}})
	}))
	defer server.Close()

	client := NewGoJudgeClient(server.URL, zap.NewNop())

	_, err := client.Execute(context.Background(), outbound.ExecutionRequest{
		Language:   "GO",
		SourceCode: "package main\nfunc main() {}\n",
		Limits: outbound.ExecutionLimits{
			TimeLimitMS: 1_000,
		},
		TestCases: []outbound.ExecutionTestCase{{
			Index: 1,
			ID:    "case-1",
			Kind:  "custom",
		}},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !sawCompile {
		t.Fatal("compile request was not observed")
	}
}
