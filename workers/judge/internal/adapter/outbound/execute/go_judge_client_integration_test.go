//go:build integration

package execute

import (
	"context"
	"os"
	"testing"
	"time"

	"go-judge-system/pkg/gojudge"
	sharedgrpc "go-judge-system/pkg/grpc"
	"go-judge-system/workers/judge/internal/application/port/outbound"

	judgepb "github.com/criyle/go-judge/pb"
	"go.uber.org/zap"
)

func TestGoJudgeClientGRPCIntegrationCompileAndRunCPP(t *testing.T) {
	address := os.Getenv("GO_JUDGE_GRPC_INTEGRATION_ADDR")
	if address == "" {
		t.Skip("set GO_JUDGE_GRPC_INTEGRATION_ADDR to run against a local go-judge sandbox")
	}

	conn, err := sharedgrpc.NewClientConn(address, sharedgrpc.WithInsecureTransport())
	if err != nil {
		t.Fatalf("create go-judge gRPC connection: %v", err)
	}
	defer func() { _ = conn.Close() }()

	client := NewGoJudgeClient(judgepb.NewExecutorClient(conn), zap.NewNop())
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	const source = `#include <bits/stdc++.h>
using namespace std;

int main() {
	long long a, b;
	cin >> a >> b;
	cout << a + b << '\n';
}`
	limits := normalizeLimits(outbound.ExecutionLimits{
		TimeLimitMS:      2_000,
		MemoryLimitKB:    256 * 1024,
		OutputLimitBytes: 1024 * 1024,
	})
	langCfg, ok := gojudge.GetLanguageConfig("CPP", gojudge.GetSourceFileName("CPP"), gojudge.GetExeFileName("CPP"))
	if !ok || langCfg.Compile == nil {
		t.Fatal("CPP compiler configuration is unavailable")
	}

	compileResult, executableFileID, err := client.compile(ctx, "CPP", source, langCfg, limits)
	if err != nil {
		t.Fatalf("compile through gRPC: %v", err)
	}
	if compileResult != nil {
		t.Fatalf("compile result = %#v, want successful compile", compileResult)
	}
	if executableFileID == "" {
		t.Fatal("compile returned an empty executable FileID")
	}
	t.Log("compile RPC accepted; executable FileID present")

	testCases := []outbound.ExecutionTestCase{
		{Index: 1, ID: "one", Stdin: "1 2\n", ExpectedOutput: stringPtr("3\n")},
		{Index: 2, ID: "two", Stdin: "100 200\n", ExpectedOutput: stringPtr("300\n")},
		{Index: 3, ID: "three", Stdin: "-5 10\n", ExpectedOutput: stringPtr("5\n")},
	}
	runResults, err := client.runBatch(ctx, "CPP", source, langCfg, limits, true, executableFileID, testCases, 0)
	if err != nil {
		t.Fatalf("run cached executable through gRPC: %v", err)
	}
	if len(runResults) != len(testCases) {
		t.Fatalf("run result count = %d, want %d", len(runResults), len(testCases))
	}
	for index, raw := range runResults {
		if raw == nil {
			t.Fatalf("run result %d is nil", index)
		}
		result := mapTestCaseResult("CPP", testCases[index], raw, false)
		if result.Status != "ACCEPTED" {
			t.Fatalf("testcase %s status = %s, want ACCEPTED", testCases[index].ID, result.Status)
		}
		if result.ActualOutput == nil || *result.ActualOutput != *testCases[index].ExpectedOutput {
			t.Fatalf("testcase %s stdout = %v, want %q", testCases[index].ID, result.ActualOutput, *testCases[index].ExpectedOutput)
		}
		t.Logf("run RPC testcase=%s status=%s stdout=%q", testCases[index].ID, result.Status, *result.ActualOutput)
	}
}
