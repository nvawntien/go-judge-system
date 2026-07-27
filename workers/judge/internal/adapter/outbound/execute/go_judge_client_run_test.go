package execute

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
			{Status: "Nonzero Exit Status", ExitStatus: 1, Files: map[string]string{"stdout": "", "stderr": "boom"}, Time: 2_000_000, Memory: 2048},
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
}

func TestRunCodeCompileErrorReturnsNoTests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(gojudge.Response{{Status: "Nonzero Exit Status", Files: map[string]string{"stderr": "undefined symbol"}}})
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
}
