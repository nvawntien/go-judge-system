//go:build integration

package execute

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"go-judge-system/pkg/gojudge"
	sharedgrpc "go-judge-system/pkg/grpc"
	"go-judge-system/workers/judge/internal/application/port/outbound"

	judgepb "github.com/criyle/go-judge/pb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
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
	runResults, err := client.runBatch(ctx, "CPP", source, langCfg, limits, true, executableFileID, nil, testCases, 0)
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

func TestGoJudgeClientGRPCIntegrationSupportedLanguages(t *testing.T) {
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
	programs := map[string]string{
		"CPP":    "#include <iostream>\nint main(){long long a,b;std::cin>>a>>b;std::cout<<a+b<<'\\n';}",
		"GO":     "package main\nimport \"fmt\"\nfunc main(){var a,b int64; fmt.Scan(&a,&b); fmt.Println(a+b)}",
		"PYTHON": "a,b=map(int,input().split())\nprint(a+b)",
		"JAVA":   "import java.util.*; public class Main { public static void main(String[] a){ Scanner s=new Scanner(System.in); System.out.println(s.nextLong()+s.nextLong()); } }",
	}
	for language, source := range programs {
		t.Run(language, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			result, err := client.Execute(ctx, outbound.ExecutionRequest{Language: language, SourceCode: source, StopOnFirstFailure: true, Limits: outbound.ExecutionLimits{TimeLimitMS: 5_000, MemoryLimitKB: 256 * 1024, OutputLimitBytes: 1024 * 1024}, TestCases: []outbound.ExecutionTestCase{{Index: 1, ID: "sum", Stdin: "1 2\n", ExpectedOutput: stringPtr("3\n")}}})
			if err != nil || result.Status != "ACCEPTED" || len(result.TestCases) != 1 || result.TestCases[0].ActualOutput == nil || *result.TestCases[0].ActualOutput != "3\n" {
				if result != nil && result.CompileOutput != nil {
					t.Logf("%s compile diagnostic: %s", language, *result.CompileOutput)
				}
				t.Fatalf("Execute(%s) result/error = %#v/%v", language, result, err)
			}
		})
	}
}

func TestGoJudgeClientGRPCIntegrationLargeOfficialLocalFile(t *testing.T) {
	address := os.Getenv("GO_JUDGE_GRPC_INTEGRATION_ADDR")
	localPath := os.Getenv("GO_JUDGE_LOCALFILE_INTEGRATION_PATH")
	if address == "" || localPath == "" {
		t.Skip("set GO_JUDGE_GRPC_INTEGRATION_ADDR and GO_JUDGE_LOCALFILE_INTEGRATION_PATH for LocalFile integration")
	}
	conn, err := sharedgrpc.NewClientConn(address, sharedgrpc.WithInsecureTransport())
	if err != nil {
		t.Fatalf("create go-judge gRPC connection: %v", err)
	}
	defer func() { _ = conn.Close() }()
	client := NewGoJudgeClient(judgepb.NewExecutorClient(conn), zap.NewNop())
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	langCfg, ok := gojudge.GetLanguageConfig("CPP", gojudge.GetSourceFileName("CPP"), gojudge.GetExeFileName("CPP"))
	if !ok || langCfg.Compile == nil {
		t.Fatal("CPP compiler configuration unavailable")
	}
	limits := normalizeLimits(outbound.ExecutionLimits{TimeLimitMS: 2_000, MemoryLimitKB: 256 * 1024, OutputLimitBytes: 1024})
	const source = `#include <iostream>
int main() { char c; long long n = 0; while (std::cin.get(c)) n++; std::cout << n << '\n'; }`
	compileResult, executableFileID, err := client.compile(ctx, "CPP", source, langCfg, limits)
	if err != nil || compileResult != nil || executableFileID == "" {
		t.Fatalf("compile result/fileID/error = %#v/%q/%v", compileResult, executableFileID, err)
	}
	defer client.cleanupExecutableFile(executableFileID)
	input := strings.Repeat("x", sandboxGRPCSafeRequestBytes+1)
	results, err := client.runBatch(ctx, "CPP", source, langCfg, limits, true, executableFileID, nil, []outbound.ExecutionTestCase{{
		Index: 1, ID: "large", Kind: "official", Stdin: input, SandboxInputPath: localPath, ExpectedOutput: stringPtr(fmt.Sprintf("%d\n", len(input))),
	}}, 0)
	if err != nil || len(results) != 1 {
		t.Fatalf("large LocalFile run results/error = %#v/%v", results, err)
	}
	result := mapTestCaseResult("CPP", outbound.ExecutionTestCase{Index: 1, ID: "large", ExpectedOutput: stringPtr(fmt.Sprintf("%d\n", len(input)))}, results[0], false)
	if result.Status != "ACCEPTED" {
		t.Fatalf("large LocalFile result = %#v", result)
	}
}

func TestGoJudgeClientGRPCIntegrationAggregateResponseOverFourMiB(t *testing.T) {
	address := os.Getenv("GO_JUDGE_GRPC_INTEGRATION_ADDR")
	if address == "" {
		t.Skip("set GO_JUDGE_GRPC_INTEGRATION_ADDR for large-response integration")
	}
	conn, err := sharedgrpc.NewClientConn(address, sharedgrpc.WithInsecureTransport(), sharedgrpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(SandboxGRPCMaxReceiveBytes)))
	if err != nil {
		t.Fatalf("create bounded large-response connection: %v", err)
	}
	defer func() { _ = conn.Close() }()
	client := NewGoJudgeClient(judgepb.NewExecutorClient(conn), zap.NewNop())
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	langCfg, _ := gojudge.GetLanguageConfig("CPP", gojudge.GetSourceFileName("CPP"), gojudge.GetExeFileName("CPP"))
	limits := normalizeLimits(outbound.ExecutionLimits{TimeLimitMS: 5_000, MemoryLimitKB: 256 * 1024, OutputLimitBytes: 1 << 20})
	compileResult, executableFileID, err := client.compile(ctx, "CPP", `#include <iostream>
int main(){ for(int i=0;i<(1<<20);i++) std::cout << 'x'; }`, langCfg, limits)
	if err != nil || compileResult != nil || executableFileID == "" {
		t.Fatalf("compile result/fileID/error = %#v/%q/%v", compileResult, executableFileID, err)
	}
	defer client.cleanupExecutableFile(executableFileID)
	expected := strings.Repeat("x", 1<<20)
	tests := make([]outbound.ExecutionTestCase, 4)
	for i := range tests {
		tests[i] = outbound.ExecutionTestCase{Index: i + 1, ID: fmt.Sprintf("large-%d", i), Stdin: "", ExpectedOutput: &expected}
	}
	results, err := client.runBatch(ctx, "CPP", "", langCfg, limits, true, executableFileID, nil, tests, 0)
	if err != nil || len(results) != 4 {
		t.Fatalf("4 MiB aggregate response results/error = %d/%v", len(results), err)
	}
	for i, result := range results {
		if got := string(result.GetFiles()["stdout"]); got != expected {
			t.Fatalf("stdout %d length=%d want=%d", i, len(got), len(expected))
		}
	}
}

// TestGoJudgeClientGRPCIntegrationOutputLimit keeps per-command OLE semantics
// separate from the Worker receive envelope tested above. It is intentionally
// a real executorserver test: a legal aggregate response may exceed 4 MiB,
// while one command exceeding its collector cap must remain OLE.
func TestGoJudgeClientGRPCIntegrationOutputLimit(t *testing.T) {
	address := os.Getenv("GO_JUDGE_GRPC_INTEGRATION_ADDR")
	if address == "" {
		t.Skip("set GO_JUDGE_GRPC_INTEGRATION_ADDR to run against a local go-judge sandbox")
	}
	conn, err := sharedgrpc.NewClientConn(address, sharedgrpc.WithInsecureTransport(), sharedgrpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(SandboxGRPCMaxReceiveBytes)))
	if err != nil {
		t.Fatalf("create go-judge gRPC connection: %v", err)
	}
	defer func() { _ = conn.Close() }()
	client := NewGoJudgeClient(judgepb.NewExecutorClient(conn), zap.NewNop())
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	langCfg, ok := gojudge.GetLanguageConfig("CPP", gojudge.GetSourceFileName("CPP"), gojudge.GetExeFileName("CPP"))
	if !ok || langCfg.Compile == nil {
		t.Fatal("CPP compiler configuration unavailable")
	}
	const outputLimit = int64(1024)
	limits := normalizeLimits(outbound.ExecutionLimits{TimeLimitMS: 2_000, MemoryLimitKB: 256 * 1024, OutputLimitBytes: outputLimit})
	compileResult, executableFileID, err := client.compile(ctx, "CPP", `#include <iostream>
int main(){ for(int i=0;i<1025;i++) std::cout << 'x'; }`, langCfg, limits)
	if err != nil || compileResult != nil || executableFileID == "" {
		t.Fatalf("compile result/fileID/error = %#v/%q/%v", compileResult, executableFileID, err)
	}
	defer client.cleanupExecutableFile(executableFileID)
	testCase := outbound.ExecutionTestCase{Index: 1, ID: "ole", Stdin: "", ExpectedOutput: stringPtr("")}
	results, err := client.runBatch(ctx, "CPP", "", langCfg, limits, true, executableFileID, nil, []outbound.ExecutionTestCase{testCase}, 0)
	if err != nil || len(results) != 1 {
		t.Fatalf("OLE run results/error = %#v/%v", results, err)
	}
	mapped := mapTestCaseResult("CPP", testCase, results[0], false)
	if mapped.Status != "OUTPUT_LIMIT_EXCEEDED" {
		t.Fatalf("OLE mapping = %#v, want OUTPUT_LIMIT_EXCEEDED", mapped)
	}
}

// TestGoJudgeClientGRPCIntegrationPersistentConnReconnect is externally
// orchestrated so the test keeps one production-style ClientConn while the
// temporary sandbox container is replaced. It is skipped unless both marker
// paths are supplied by the local Docker proof harness.
func TestGoJudgeClientGRPCIntegrationPersistentConnReconnect(t *testing.T) {
	address, ready, resume := os.Getenv("GO_JUDGE_GRPC_INTEGRATION_ADDR"), os.Getenv("GO_JUDGE_RECONNECT_READY"), os.Getenv("GO_JUDGE_RECONNECT_RESUME")
	if address == "" || ready == "" || resume == "" {
		t.Skip("set GO_JUDGE_GRPC_INTEGRATION_ADDR, GO_JUDGE_RECONNECT_READY, and GO_JUDGE_RECONNECT_RESUME")
	}
	conn, err := sharedgrpc.NewClientConn(address, sharedgrpc.WithInsecureTransport())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := NewGoJudgeClient(judgepb.NewExecutorClient(conn), zap.NewNop())
	execOnce := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		result, err := client.Execute(ctx, outbound.ExecutionRequest{Language: "PYTHON", SourceCode: "print('ok')", TestCases: []outbound.ExecutionTestCase{{Index: 1, ID: "case", ExpectedOutput: stringPtr("ok\n")}}})
		if err != nil || result.Status != "ACCEPTED" {
			t.Fatalf("persistent-connection Exec=%#v/%v", result, err)
		}
	}
	execOnce()
	if err := os.WriteFile(ready, []byte("ready\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		if _, err := os.Stat(resume); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for sandbox replacement")
		}
		time.Sleep(50 * time.Millisecond)
	}
	// NewClientConn is deliberately not called here. grpc-go reconnects the
	// original connection through Docker DNS after the replacement is ready.
	for deadline := time.Now().Add(20 * time.Second); ; {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		result, callErr := client.Execute(ctx, outbound.ExecutionRequest{Language: "PYTHON", SourceCode: "print('ok')", TestCases: []outbound.ExecutionTestCase{{Index: 1, ID: "case", ExpectedOutput: stringPtr("ok\n")}}})
		cancel()
		if callErr == nil && result.Status == "ACCEPTED" {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("same ClientConn did not recover: result=%#v err=%v", result, callErr)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestGoJudgeClientGRPCIntegrationTestcaseInputCache(t *testing.T) {
	address := os.Getenv("GO_JUDGE_GRPC_INTEGRATION_ADDR")
	if address == "" {
		t.Skip("set GO_JUDGE_GRPC_INTEGRATION_ADDR to run against a local go-judge sandbox")
	}

	conn, err := sharedgrpc.NewClientConn(address, sharedgrpc.WithInsecureTransport())
	if err != nil {
		t.Fatalf("create go-judge gRPC connection: %v", err)
	}
	defer func() { _ = conn.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	sandboxClient := judgepb.NewExecutorClient(conn)
	client := NewGoJudgeClient(sandboxClient, zap.NewNop(), enabledTestcaseCacheConfig())
	limits := normalizeLimits(outbound.ExecutionLimits{TimeLimitMS: 2_000, MemoryLimitKB: 256 * 1024, OutputLimitBytes: 1024 * 1024})
	langCfg, ok := gojudge.GetLanguageConfig("CPP", gojudge.GetSourceFileName("CPP"), gojudge.GetExeFileName("CPP"))
	if !ok || langCfg.Compile == nil {
		t.Fatal("CPP compiler configuration is unavailable")
	}
	const source = `#include <bits/stdc++.h>
using namespace std;
int main() { long long a, b; cin >> a >> b; cout << a + b << '\n'; }`
	compileResult, executableFileID, err := client.compile(ctx, "CPP", source, langCfg, limits)
	if err != nil || compileResult != nil || executableFileID == "" {
		t.Fatalf("compile result/fileID/error = %#v/%q/%v", compileResult, executableFileID, err)
	}
	checksum := sha256.Sum256([]byte(fmt.Sprintf("go-judge-testcase-cache-integration-%d", time.Now().UTC().UnixNano())))
	identity := &outbound.TestcaseDatasetIdentity{ProblemID: 9_999_991, Version: 1, DatasetChecksum: hex.EncodeToString(checksum[:])}
	expected := "3\n"
	testCases := []outbound.ExecutionTestCase{{Index: 1, ID: "cache-case-1", Kind: "official", Stdin: "1 2\n", ExpectedOutput: &expected}}
	key, ok := newTestcaseCacheKey(identity, testCases[0])
	if !ok {
		t.Fatal("build testcase cache key")
	}

	before, err := sandboxClient.FileList(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("FileList before cache population: %v", err)
	}
	baselineEntries := countSandboxTestcaseCacheEntries(before)

	first, err := client.runBatch(ctx, "CPP", source, langCfg, limits, true, executableFileID, identity, testCases, 0)
	assertIntegrationAccepted(t, testCases, first, err)
	fileID, found := client.testcaseCache.lookup(key)
	if !found || fileID == "" {
		t.Fatal("first execution did not retain sandbox testcase FileID")
	}
	afterFirst, err := sandboxClient.FileList(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("FileList after first execution: %v", err)
	}
	if got, want := countSandboxTestcaseCacheEntries(afterFirst), baselineEntries+1; got != want {
		t.Fatalf("sandbox testcase cache entries after first execution = %d, want baseline+1=%d", got, want)
	}

	second, err := client.runBatch(ctx, "CPP", source, langCfg, limits, true, executableFileID, identity, testCases, 1)
	assertIntegrationAccepted(t, testCases, second, err)
	if got, _ := client.testcaseCache.lookup(key); got != fileID {
		t.Fatalf("warm cache FileID = %q, want %q", got, fileID)
	}
	afterSecond, err := sandboxClient.FileList(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("FileList after warm execution: %v", err)
	}
	if got, want := countSandboxTestcaseCacheEntries(afterSecond), baselineEntries+1; got != want {
		t.Fatalf("sandbox testcase cache entries after warm execution = %d, want baseline+1=%d", got, want)
	}

	// Simulate a Worker restart: a new Worker-side index must rebuild only from
	// the AstraCode namespace and reuse the existing sandbox file.
	restartedWorker := NewGoJudgeClient(sandboxClient, zap.NewNop(), enabledTestcaseCacheConfig())
	reconciled, err := restartedWorker.runBatch(ctx, "CPP", source, langCfg, limits, true, executableFileID, identity, testCases, 2)
	assertIntegrationAccepted(t, testCases, reconciled, err)
	if got, _ := restartedWorker.testcaseCache.lookup(key); got != fileID {
		t.Fatalf("reconciled FileID = %q, want %q", got, fileID)
	}
	afterRestart, err := sandboxClient.FileList(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("FileList after Worker restart simulation: %v", err)
	}
	if got, want := countSandboxTestcaseCacheEntries(afterRestart), baselineEntries+1; got != want {
		t.Fatalf("sandbox testcase cache entries after Worker restart = %d, want baseline+1=%d", got, want)
	}

	// Force a controlled lifecycle eviction only for the known AstraCode
	// testcase entry. The compile FileID is deliberately not in this index.
	restartedWorker.testcaseCache.mu.Lock()
	restartedWorker.testcaseCache.cfg.MaxBytes = 1
	restartedWorker.testcaseCache.nextCleanup = time.Now().Add(time.Hour)
	restartedWorker.testcaseCache.mu.Unlock()
	restartedWorker.testcaseCache.cleanup(ctx)
	afterEviction, err := sandboxClient.FileList(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("FileList after controlled eviction: %v", err)
	}
	if got, want := countSandboxTestcaseCacheEntries(afterEviction), baselineEntries; got != want {
		t.Fatalf("sandbox testcase cache entries after controlled eviction = %d, want baseline=%d", got, want)
	}
	if _, found := restartedWorker.testcaseCache.lookup(key); found {
		t.Fatal("controlled lifecycle eviction retained the testcase FileID")
	}
	// Restore the valid configured budget before testing repopulation. The
	// one-byte value above exists solely to force this controlled eviction.
	restartedWorker.testcaseCache.mu.Lock()
	restartedWorker.testcaseCache.cfg.MaxBytes = enabledTestcaseCacheConfig().MaxBytes
	restartedWorker.testcaseCache.mu.Unlock()

	// Re-execution must repopulate the testcase cache while continuing to use
	// the original compiled executable FileID successfully.
	afterRepopulate, err := restartedWorker.runBatch(ctx, "CPP", source, langCfg, limits, true, executableFileID, identity, testCases, 3)
	assertIntegrationAccepted(t, testCases, afterRepopulate, err)
	newFileID, found := restartedWorker.testcaseCache.lookup(key)
	if !found || newFileID == "" || newFileID == fileID {
		t.Fatalf("repopulated testcase FileID = %q/%t, want new non-empty FileID distinct from %q", newFileID, found, fileID)
	}

	// Submission-scoped executable cleanup must remove only the compile output;
	// the independently owned testcase cache entry remains available.
	beforeExecutableCleanup, err := sandboxClient.FileList(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("FileList before executable cleanup: %v", err)
	}
	if _, found := beforeExecutableCleanup.GetFileIDs()[executableFileID]; !found {
		t.Fatal("compiled executable FileID was absent before explicit cleanup")
	}
	restartedWorker.cleanupExecutableFile(executableFileID)
	afterExecutableCleanup, err := sandboxClient.FileList(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("FileList after executable cleanup: %v", err)
	}
	if _, found := afterExecutableCleanup.GetFileIDs()[executableFileID]; found {
		t.Fatal("compiled executable FileID remained after cleanup")
	}
	if _, found := afterExecutableCleanup.GetFileIDs()[newFileID]; !found {
		t.Fatal("executable cleanup removed the testcase-cache FileID")
	}
	cachedInput, err := sandboxClient.FileGet(ctx, &judgepb.FileID{FileID: newFileID})
	if err != nil || string(cachedInput.GetContent()) != testCases[0].Stdin {
		t.Fatalf("testcase-cache FileID after executable cleanup = %q/%v", string(cachedInput.GetContent()), err)
	}
}

func countSandboxTestcaseCacheEntries(list *judgepb.FileListType) int {
	count := 0
	for _, name := range list.GetFileIDs() {
		if _, _, _, ok := parseTestcaseCacheName(name); ok {
			count++
		}
	}
	return count
}

func assertIntegrationAccepted(t *testing.T, testCases []outbound.ExecutionTestCase, results []*judgepb.Response_Result, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("run cached testcase: %v", err)
	}
	if len(results) != len(testCases) {
		t.Fatalf("result count = %d, want %d", len(results), len(testCases))
	}
	for index, result := range results {
		mapped := mapTestCaseResult("CPP", testCases[index], result, true)
		if mapped.Status != "ACCEPTED" || mapped.ActualOutput == nil || *mapped.ActualOutput != *testCases[index].ExpectedOutput {
			t.Fatalf("testcase %d result = %#v", index, mapped)
		}
	}
}
