//go:build integration

package execute

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"go-judge-system/pkg/gojudge"
	sharedgrpc "go-judge-system/pkg/grpc"
	"go-judge-system/workers/judge/internal/application/port/outbound"

	judgepb "github.com/criyle/go-judge/pb"
	"go.uber.org/zap"
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
