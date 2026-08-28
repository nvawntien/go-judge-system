//go:build integration

package execute

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-judge-system/pkg/gojudge"
	sharedgrpc "go-judge-system/pkg/grpc"
	"go-judge-system/workers/judge/internal/application/port/outbound"
	workerdomain "go-judge-system/workers/judge/internal/domain"

	judgepb "github.com/criyle/go-judge/pb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	testcaseCacheABWarmupIterations = 10
	testcaseCacheABMeasurements     = 100
	testcaseCacheABBootstrapSamples = 10_000
	testcaseCacheABBootstrapSeed    = int64(20_260_828)
)

// These payloads deliberately keep a four-command MemoryFile batch below the
// default 4 MiB gRPC message limit. 896 KiB is the largest useful "about 1
// MiB" class without changing either the Worker or executorserver limits.
var testcaseCacheABSizeClasses = []testcaseCacheABSizeClass{
	{Name: "small", BytesPerTestcase: 1 * 1024},
	{Name: "medium", BytesPerTestcase: 64 * 1024},
	{Name: "large", BytesPerTestcase: 896 * 1024},
}

type testcaseCacheABSizeClass struct {
	Name             string
	BytesPerTestcase int
}

// TestGoJudgeTestcaseCacheAB is an explicit local integration benchmark. It
// compares the current production GoJudgeClient's nil-dataset MemoryFile path
// with its production TestcaseDataset CachedFile path. It needs a local
// executorserver gRPC address and never runs as part of ordinary go test.
func TestGoJudgeTestcaseCacheAB(t *testing.T) {
	address := testcaseCacheABRequiredEnv(t, "GO_JUDGE_GRPC_INTEGRATION_ADDR")
	resultsDir := testcaseCacheABResultsDir(t)
	if err := os.MkdirAll(resultsDir, 0o700); err != nil {
		t.Fatalf("create benchmark artifact directory: %v", err)
	}

	conn, err := sharedgrpc.NewClientConn(address, sharedgrpc.WithInsecureTransport())
	if err != nil {
		t.Fatalf("create go-judge gRPC connection: %v", err)
	}
	defer func() { _ = conn.Close() }()

	rpc := &testcaseCacheABRPC{ExecutorClient: judgepb.NewExecutorClient(conn)}
	client := NewGoJudgeClient(rpc, zap.NewNop(), enabledTestcaseCacheConfig())
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	limits := normalizeLimits(outbound.ExecutionLimits{
		TimeLimitMS:      2_000,
		MemoryLimitKB:    256 * 1024,
		OutputLimitBytes: 1024 * 1024,
	})
	langCfg, ok := gojudge.GetLanguageConfig("CPP", gojudge.GetSourceFileName("CPP"), gojudge.GetExeFileName("CPP"))
	if !ok || langCfg.Compile == nil {
		t.Fatal("CPP compiler configuration is unavailable")
	}

	compileStarted := time.Now()
	compileResult, executableFileID, err := client.compile(ctx, "CPP", testcaseCacheABSource, langCfg, limits)
	compileDuration := time.Since(compileStarted)
	if err != nil {
		t.Fatalf("compile benchmark program through actual GoJudgeClient: %v", err)
	}
	if compileResult != nil || executableFileID == "" {
		t.Fatalf("compile result/file ID = %#v/%q, want success and executable FileID", compileResult, executableFileID)
	}

	resources := newTestcaseCacheABResourceSampler(os.Getenv("GO_JUDGE_TESTCASE_CACHE_AB_CONTAINER"))
	resources.Start(ctx)
	defer resources.Stop()

	records := make([]testcaseCacheABRecord, 0, len(testcaseCacheABSizeClasses)*testcaseCacheABMeasurements*2)
	cold := make([]testcaseCacheABColdResult, 0, len(testcaseCacheABSizeClasses))
	runNonce := time.Now().UTC().UnixNano()
	for classIndex, sizeClass := range testcaseCacheABSizeClasses {
		testCases := testcaseCacheABSyntheticTestCases(sizeClass.BytesPerTestcase)
		identity := testcaseCacheABDatasetIdentity(classIndex, runNonce)
		assertTestcaseCacheABIdentity(t, identity, testCases)

		beforeCold := rpc.snapshotFileAdds()
		populationStarted := time.Now()
		for _, testCase := range testCases {
			if _, cached, err := client.testcaseCache.getOrAdd(ctx, identity, testCase); err != nil {
				t.Fatalf("%s cold cache population: %v", sizeClass.Name, err)
			} else if !cached {
				t.Fatalf("%s cold cache population unexpectedly used MemoryFile fallback", sizeClass.Name)
			}
		}
		populationDuration := time.Since(populationStarted)
		afterPopulation := rpc.snapshotFileAdds()
		firstCached, err := testcaseCacheABRun(ctx, client, "cached_file", sizeClass, testcaseCacheABSource, langCfg, limits, executableFileID, identity, testCases, 0)
		if err != nil {
			_ = testcaseCacheABWriteArtifacts(resultsDir, compileDuration, records, cold, resources.Samples())
			t.Fatalf("%s first CachedFile execution after population: %v", sizeClass.Name, err)
		}
		afterFirstExecution := rpc.snapshotFileAdds()
		if afterFirstExecution != afterPopulation {
			t.Fatalf("%s first CachedFile execution populated files: before=%+v after=%+v", sizeClass.Name, afterPopulation, afterFirstExecution)
		}
		cold = append(cold, testcaseCacheABColdResult{
			InputSizeClass:     sizeClass.Name,
			BytesPerTestcase:   sizeClass.BytesPerTestcase,
			TestcaseCount:      len(testCases),
			FileAddCalls:       afterPopulation.Calls - beforeCold.Calls,
			BytesUploaded:      afterPopulation.Bytes - beforeCold.Bytes,
			PopulationDuration: populationDuration,
			FirstExecutionWall: firstCached.TotalWall,
		})
		if got := afterPopulation.Calls - beforeCold.Calls; got != int64(len(testCases)) {
			t.Fatalf("%s cold FileAdd calls = %d, want %d", sizeClass.Name, got, len(testCases))
		}

		for i := 0; i < testcaseCacheABWarmupIterations; i++ {
			if _, err := testcaseCacheABRun(ctx, client, "memory_file", sizeClass, testcaseCacheABSource, langCfg, limits, executableFileID, nil, testCases, -i-1); err != nil {
				t.Fatalf("%s MemoryFile warmup %d: %v", sizeClass.Name, i+1, err)
			}
			if _, err := testcaseCacheABRun(ctx, client, "cached_file", sizeClass, testcaseCacheABSource, langCfg, limits, executableFileID, identity, testCases, -i-1); err != nil {
				t.Fatalf("%s CachedFile warmup %d: %v", sizeClass.Name, i+1, err)
			}
		}

		order := make([]string, 0, testcaseCacheABMeasurements*2)
		for i := 0; i < testcaseCacheABMeasurements; i++ {
			order = append(order, "memory_file", "cached_file")
		}
		rng := rand.New(rand.NewSource(testcaseCacheABBootstrapSeed + int64(classIndex)))
		rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
		iterations := map[string]int{"memory_file": 0, "cached_file": 0}

		for _, mode := range order {
			iterations[mode]++
			var dataset *outbound.TestcaseDatasetIdentity
			if mode == "cached_file" {
				dataset = identity
			}
			before := rpc.snapshotFileAdds()
			record, runErr := testcaseCacheABRun(ctx, client, mode, sizeClass, testcaseCacheABSource, langCfg, limits, executableFileID, dataset, testCases, iterations[mode])
			after := rpc.snapshotFileAdds()
			record.FileAddCalls = after.Calls - before.Calls
			record.FileAddBytes = after.Bytes - before.Bytes
			if runErr != nil {
				record.TransportError = runErr.Error()
			}
			records = append(records, record)
			if mode == "cached_file" && record.FileAddCalls != 0 {
				_ = testcaseCacheABWriteArtifacts(resultsDir, compileDuration, records, cold, resources.Samples())
				t.Fatalf("%s hot CachedFile iteration %d populated %d files; benchmark is invalid", sizeClass.Name, record.Iteration, record.FileAddCalls)
			}
			if runErr != nil {
				_ = testcaseCacheABWriteArtifacts(resultsDir, compileDuration, records, cold, resources.Samples())
				t.Fatalf("%s %s measurement %d: %v", sizeClass.Name, mode, record.Iteration, runErr)
			}
		}
	}

	resources.Stop()
	if err := testcaseCacheABWriteArtifacts(resultsDir, compileDuration, records, cold, resources.Samples()); err != nil {
		t.Fatalf("write benchmark artifacts: %v", err)
	}
	t.Logf("testcase-cache A/B complete: %s", resultsDir)
}

const testcaseCacheABSource = `#include <bits/stdc++.h>
using namespace std;

int main() {
	string input((istreambuf_iterator<char>(cin)), istreambuf_iterator<char>());
	uint64_t hash = 1469598103934665603ULL;
	for (unsigned char c : input) {
		hash ^= c;
		hash *= 1099511628211ULL;
	}
	cout << hash << '\n';
}`

type testcaseCacheABRPC struct {
	judgepb.ExecutorClient
	fileAddCalls atomic.Int64
	fileAddBytes atomic.Int64
}

func (c *testcaseCacheABRPC) Exec(ctx context.Context, request *judgepb.Request, options ...grpc.CallOption) (*judgepb.Response, error) {
	return c.ExecutorClient.Exec(ctx, request, options...)
}

func (c *testcaseCacheABRPC) FileAdd(ctx context.Context, file *judgepb.FileContent, options ...grpc.CallOption) (*judgepb.FileID, error) {
	c.fileAddCalls.Add(1)
	c.fileAddBytes.Add(int64(len(file.GetContent())))
	return c.ExecutorClient.FileAdd(ctx, file, options...)
}

func (c *testcaseCacheABRPC) FileList(ctx context.Context, request *emptypb.Empty, options ...grpc.CallOption) (*judgepb.FileListType, error) {
	return c.ExecutorClient.FileList(ctx, request, options...)
}

type testcaseCacheABFileAddSnapshot struct{ Calls, Bytes int64 }

func (c *testcaseCacheABRPC) snapshotFileAdds() testcaseCacheABFileAddSnapshot {
	return testcaseCacheABFileAddSnapshot{Calls: c.fileAddCalls.Load(), Bytes: c.fileAddBytes.Load()}
}

type testcaseCacheABRecord struct {
	TimestampUTC       time.Time
	Mode               string
	InputSizeClass     string
	BytesPerTestcase   int
	TotalInputBytes    int
	Iteration          int
	TotalWall          time.Duration
	BatchWall          []time.Duration
	BatchCommandCounts []int
	BatchResultCounts  []int
	FinalVerdict       string
	Correct            bool
	TransportError     string
	MalformedResponse  bool
	FileAddCalls       int64
	FileAddBytes       int64
}

func testcaseCacheABRun(ctx context.Context, client *GoJudgeClient, mode string, sizeClass testcaseCacheABSizeClass, source string, langCfg *gojudge.LanguageConfig, limits outbound.ExecutionLimits, executableFileID string, identity *outbound.TestcaseDatasetIdentity, testCases []outbound.ExecutionTestCase, iteration int) (testcaseCacheABRecord, error) {
	record := testcaseCacheABRecord{
		TimestampUTC:     time.Now().UTC(),
		Mode:             mode,
		InputSizeClass:   sizeClass.Name,
		BytesPerTestcase: sizeClass.BytesPerTestcase,
		TotalInputBytes:  len(testCases) * sizeClass.BytesPerTestcase,
		Iteration:        iteration,
		FinalVerdict:     "ACCEPTED",
	}
	started := time.Now()
	for start, batchIndex := 0, 0; start < len(testCases); start, batchIndex = start+officialBatchSize, batchIndex+1 {
		end := min(start+officialBatchSize, len(testCases))
		batch := testCases[start:end]
		batchStarted := time.Now()
		results, err := client.runBatch(ctx, "CPP", source, langCfg, limits, true, executableFileID, identity, batch, batchIndex)
		record.BatchWall = append(record.BatchWall, time.Since(batchStarted))
		record.BatchCommandCounts = append(record.BatchCommandCounts, len(batch))
		record.BatchResultCounts = append(record.BatchResultCounts, len(results))
		if err != nil {
			record.TotalWall = time.Since(started)
			return record, err
		}
		if err := testcaseCacheABValidateResults(batch, results); err != nil {
			record.MalformedResponse = len(results) != len(batch)
			record.TotalWall = time.Since(started)
			return record, err
		}
	}
	record.TotalWall = time.Since(started)
	record.Correct = true
	return record, nil
}

func testcaseCacheABValidateResults(testCases []outbound.ExecutionTestCase, results []*judgepb.Response_Result) error {
	if len(results) != len(testCases) {
		return fmt.Errorf("returned %d results for %d testcases", len(results), len(testCases))
	}
	for i, result := range results {
		if result == nil {
			return fmt.Errorf("nil result for testcase %d", testCases[i].Index)
		}
		mapped := mapTestCaseResult("CPP", testCases[i], result, true)
		if mapped.Status != "ACCEPTED" {
			return fmt.Errorf("testcase %d verdict = %s", testCases[i].Index, mapped.Status)
		}
		if mapped.ActualOutput == nil || testCases[i].ExpectedOutput == nil || !workerdomain.OutputEqual(*mapped.ActualOutput, *testCases[i].ExpectedOutput) {
			return fmt.Errorf("testcase %d stdout did not match expected output", testCases[i].Index)
		}
	}
	return nil
}

func testcaseCacheABSyntheticTestCases(bytesPerTestcase int) []outbound.ExecutionTestCase {
	testCases := make([]outbound.ExecutionTestCase, 0, 14)
	for index := 0; index < 14; index++ {
		input := make([]byte, bytesPerTestcase)
		for offset := range input {
			input[offset] = byte('a' + (index*17+offset)%26)
		}
		expected := strconv.FormatUint(testcaseCacheABFNV1a64(input), 10) + "\n"
		testCases = append(testCases, outbound.ExecutionTestCase{
			Index: index + 1, ID: fmt.Sprintf("case-%02d", index+1), Kind: "official", Stdin: string(input), ExpectedOutput: &expected,
		})
	}
	return testCases
}

func testcaseCacheABFNV1a64(input []byte) uint64 {
	var hash uint64 = 1469598103934665603
	for _, value := range input {
		hash ^= uint64(value)
		hash *= 1099511628211
	}
	return hash
}

func testcaseCacheABDatasetIdentity(index int, runNonce int64) *outbound.TestcaseDatasetIdentity {
	// A run nonce makes cold-cache samples independent from an earlier local
	// benchmark invocation against the same temporary sandbox. It remains
	// constant for both modes and every iteration within this benchmark run.
	checksum := sha256.Sum256([]byte(fmt.Sprintf("testcase-cache-ab-dataset-%d-run-%d", index, runNonce)))
	return &outbound.TestcaseDatasetIdentity{ProblemID: 9_998_000 + int64(index), Version: 1, DatasetChecksum: hex.EncodeToString(checksum[:])}
}

func assertTestcaseCacheABIdentity(t *testing.T, identity *outbound.TestcaseDatasetIdentity, cases []outbound.ExecutionTestCase) {
	t.Helper()
	first, ok := newTestcaseCacheKey(identity, cases[0])
	if !ok {
		t.Fatal("build testcase cache identity")
	}
	if second, ok := newTestcaseCacheKey(identity, cases[0]); !ok || second != first {
		t.Fatal("identical testcase provenance did not create an identical cache key")
	}
	altered := cases[0]
	altered.Stdin += "x"
	if different, ok := newTestcaseCacheKey(identity, altered); !ok || different == first {
		t.Fatal("altered testcase input reused the original cache key")
	}
}

type testcaseCacheABColdResult struct {
	InputSizeClass     string        `json:"input_size_class"`
	BytesPerTestcase   int           `json:"bytes_per_testcase"`
	TestcaseCount      int           `json:"testcase_count"`
	FileAddCalls       int64         `json:"file_add_calls"`
	BytesUploaded      int64         `json:"bytes_uploaded"`
	PopulationDuration time.Duration `json:"-"`
	FirstExecutionWall time.Duration `json:"-"`
}

type testcaseCacheABColdJSON struct {
	InputSizeClass     string  `json:"input_size_class"`
	BytesPerTestcase   int     `json:"bytes_per_testcase"`
	TestcaseCount      int     `json:"testcase_count"`
	FileAddCalls       int64   `json:"file_add_calls"`
	BytesUploaded      int64   `json:"bytes_uploaded"`
	PopulationDuration float64 `json:"population_duration_ms"`
	FirstExecutionWall float64 `json:"first_cached_execution_wall_ms"`
	ColdEndToEnd       float64 `json:"file_add_plus_first_cached_execution_ms"`
}

type testcaseCacheABDelta struct {
	MemoryFileMS   float64 `json:"memory_file_ms"`
	CachedFileMS   float64 `json:"cached_file_ms"`
	AbsoluteMS     float64 `json:"absolute_delta_ms"`
	ImprovementPct float64 `json:"improvement_pct"`
}

type testcaseCacheABCI struct {
	Method          string  `json:"method"`
	Seed            int64   `json:"seed"`
	Resamples       int     `json:"resamples"`
	MedianDeltaMS   float64 `json:"median_delta_ms"`
	MedianLower95MS float64 `json:"median_lower_95_ms"`
	MedianUpper95MS float64 `json:"median_upper_95_ms"`
	MeanDeltaMS     float64 `json:"mean_delta_ms"`
	MeanLower95MS   float64 `json:"mean_lower_95_ms"`
	MeanUpper95MS   float64 `json:"mean_upper_95_ms"`
}

type testcaseCacheABClassSummary struct {
	InputSizeClass   string                          `json:"input_size_class"`
	BytesPerTestcase int                             `json:"bytes_per_testcase"`
	TotalInputBytes  int                             `json:"total_input_bytes_per_submission"`
	MemoryFile       transportABTransportSummary     `json:"memory_file"`
	CachedFile       transportABTransportSummary     `json:"cached_file"`
	Delta            map[string]testcaseCacheABDelta `json:"memory_file_minus_cached_file"`
	MedianCI         testcaseCacheABCI               `json:"bootstrap_95_ci"`
	Assessment       string                          `json:"local_effect_assessment"`
}

type testcaseCacheABSummary struct {
	SchemaVersion int                           `json:"schema_version"`
	GeneratedAt   time.Time                     `json:"generated_at_utc"`
	Benchmark     string                        `json:"benchmark"`
	Workload      map[string]any                `json:"workload"`
	Compile       map[string]any                `json:"compile_diagnostic_excluded_from_kpi"`
	Warmup        int                           `json:"warmup_iterations_per_mode"`
	Measurements  int                           `json:"measurement_iterations_per_mode"`
	ColdCache     []testcaseCacheABColdJSON     `json:"cold_cache"`
	Classes       []testcaseCacheABClassSummary `json:"classes"`
	Correctness   map[string]any                `json:"correctness"`
	Resources     map[string]any                `json:"resource_observation"`
}

func testcaseCacheABWriteArtifacts(resultsDir string, compileDuration time.Duration, records []testcaseCacheABRecord, cold []testcaseCacheABColdResult, resources []testcaseCacheABResourceSample) error {
	if err := testcaseCacheABWriteRawCSV(filepath.Join(resultsDir, "raw.csv"), records); err != nil {
		return err
	}
	if err := testcaseCacheABWriteResourceCSV(filepath.Join(resultsDir, "resource_samples.csv"), resources); err != nil {
		return err
	}
	summary, err := testcaseCacheABMakeSummary(compileDuration, records, cold, resources)
	if err != nil {
		return err
	}
	body, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal summary: %w", err)
	}
	if err := os.WriteFile(filepath.Join(resultsDir, "summary.json"), append(body, '\n'), 0o600); err != nil {
		return fmt.Errorf("write summary.json: %w", err)
	}
	if err := os.WriteFile(filepath.Join(resultsDir, "report.md"), []byte(testcaseCacheABReport(summary)), 0o600); err != nil {
		return fmt.Errorf("write report.md: %w", err)
	}
	return nil
}

func testcaseCacheABWriteRawCSV(path string, records []testcaseCacheABRecord) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open raw.csv: %w", err)
	}
	defer func() { _ = file.Close() }()
	writer := csv.NewWriter(file)
	if err := writer.Write([]string{"timestamp_utc", "mode", "input_size_class", "bytes_per_testcase", "total_input_bytes", "iteration", "total_testcase_wall_ms", "batch_wall_ms", "batch_command_counts", "batch_result_counts", "final_verdict", "correct", "transport_error", "malformed_response", "file_add_calls", "file_add_bytes"}); err != nil {
		return err
	}
	for _, record := range records {
		if err := writer.Write([]string{record.TimestampUTC.Format(time.RFC3339Nano), record.Mode, record.InputSizeClass, strconv.Itoa(record.BytesPerTestcase), strconv.Itoa(record.TotalInputBytes), strconv.Itoa(record.Iteration), transportABFormatMS(record.TotalWall), transportABDurationList(record.BatchWall), transportABIntList(record.BatchCommandCounts), transportABIntList(record.BatchResultCounts), record.FinalVerdict, strconv.FormatBool(record.Correct), record.TransportError, strconv.FormatBool(record.MalformedResponse), strconv.FormatInt(record.FileAddCalls, 10), strconv.FormatInt(record.FileAddBytes, 10)}); err != nil {
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("write raw.csv: %w", err)
	}
	return file.Sync()
}

func testcaseCacheABMakeSummary(compileDuration time.Duration, records []testcaseCacheABRecord, cold []testcaseCacheABColdResult, resources []testcaseCacheABResourceSample) (testcaseCacheABSummary, error) {
	summary := testcaseCacheABSummary{SchemaVersion: 1, GeneratedAt: time.Now().UTC(), Benchmark: "gRPC MemoryFile vs production sandbox testcase CachedFile Judge Core A/B", Workload: map[string]any{"language": "CPP", "synthetic": true, "testcase_count": 14, "official_batch_size": officialBatchSize, "batch_command_counts": []int{4, 4, 4, 2}, "concurrency": 1, "same_compiled_executable_file_id": true, "large_class_note": "896 KiB per testcase keeps a four-command MemoryFile batch below the default 4 MiB gRPC message limit"}, Compile: map[string]any{"transport": "grpc", "duration_ms": float64(compileDuration) / float64(time.Millisecond), "included_in_judge_core_kpi": false}, Warmup: testcaseCacheABWarmupIterations, Measurements: testcaseCacheABMeasurements, Correctness: map[string]any{"all_measured_iterations_valid": true, "correctness_failures": 0, "hot_cached_file_add_calls": 0}}
	for _, item := range cold {
		populationMS := float64(item.PopulationDuration) / float64(time.Millisecond)
		firstExecutionMS := float64(item.FirstExecutionWall) / float64(time.Millisecond)
		summary.ColdCache = append(summary.ColdCache, testcaseCacheABColdJSON{InputSizeClass: item.InputSizeClass, BytesPerTestcase: item.BytesPerTestcase, TestcaseCount: item.TestcaseCount, FileAddCalls: item.FileAddCalls, BytesUploaded: item.BytesUploaded, PopulationDuration: populationMS, FirstExecutionWall: firstExecutionMS, ColdEndToEnd: populationMS + firstExecutionMS})
	}
	for _, sizeClass := range testcaseCacheABSizeClasses {
		byMode := map[string][]testcaseCacheABRecord{"memory_file": nil, "cached_file": nil}
		for _, record := range records {
			if record.InputSizeClass != sizeClass.Name {
				continue
			}
			if record.TransportError != "" || !record.Correct || record.FinalVerdict != "ACCEPTED" {
				return testcaseCacheABSummary{}, fmt.Errorf("invalid %s %s measurement %d", sizeClass.Name, record.Mode, record.Iteration)
			}
			byMode[record.Mode] = append(byMode[record.Mode], record)
		}
		if len(byMode["memory_file"]) != testcaseCacheABMeasurements || len(byMode["cached_file"]) != testcaseCacheABMeasurements {
			return testcaseCacheABSummary{}, fmt.Errorf("%s measurement counts memory=%d cached=%d", sizeClass.Name, len(byMode["memory_file"]), len(byMode["cached_file"]))
		}
		memoryRecords := testcaseCacheABAsTransportRecords(byMode["memory_file"])
		cachedRecords := testcaseCacheABAsTransportRecords(byMode["cached_file"])
		memory := transportABTransportStats(memoryRecords)
		cached := transportABTransportStats(cachedRecords)
		memoryValues := transportABTotals(memoryRecords)
		cachedValues := transportABTotals(cachedRecords)
		deltas := map[string]testcaseCacheABDelta{"mean": testcaseCacheABMakeDelta(memory.TotalTestcaseWall.Mean, cached.TotalTestcaseWall.Mean), "p50": testcaseCacheABMakeDelta(memory.TotalTestcaseWall.P50, cached.TotalTestcaseWall.P50), "p95": testcaseCacheABMakeDelta(memory.TotalTestcaseWall.P95, cached.TotalTestcaseWall.P95), "p99": testcaseCacheABMakeDelta(memory.TotalTestcaseWall.P99, cached.TotalTestcaseWall.P99)}
		ci := testcaseCacheABBootstrapCI(memoryValues, cachedValues)
		summary.Classes = append(summary.Classes, testcaseCacheABClassSummary{InputSizeClass: sizeClass.Name, BytesPerTestcase: sizeClass.BytesPerTestcase, TotalInputBytes: sizeClass.BytesPerTestcase * 14, MemoryFile: memory, CachedFile: cached, Delta: deltas, MedianCI: ci, Assessment: testcaseCacheABAssessment(deltas["p50"], ci)})
	}
	summary.Resources = testcaseCacheABResourceSummary(resources)
	return summary, nil
}

func testcaseCacheABAsTransportRecords(records []testcaseCacheABRecord) []transportABRecord {
	converted := make([]transportABRecord, 0, len(records))
	for _, record := range records {
		converted = append(converted, transportABRecord{
			TimestampUTC:       record.TimestampUTC,
			Transport:          record.Mode,
			Iteration:          record.Iteration,
			TotalWall:          record.TotalWall,
			BatchWall:          record.BatchWall,
			BatchCommandCounts: record.BatchCommandCounts,
			BatchResultCounts:  record.BatchResultCounts,
			FinalVerdict:       record.FinalVerdict,
			TransportError:     record.TransportError,
			MalformedResponse:  record.MalformedResponse,
		})
	}
	return converted
}

func testcaseCacheABMakeDelta(memoryMS, cachedMS float64) testcaseCacheABDelta {
	delta := testcaseCacheABDelta{MemoryFileMS: memoryMS, CachedFileMS: cachedMS, AbsoluteMS: memoryMS - cachedMS}
	if memoryMS != 0 {
		delta.ImprovementPct = delta.AbsoluteMS / memoryMS * 100
	}
	return delta
}

func testcaseCacheABBootstrapCI(memory, cached []float64) testcaseCacheABCI {
	rng := rand.New(rand.NewSource(testcaseCacheABBootstrapSeed))
	medianDeltas := make([]float64, testcaseCacheABBootstrapSamples)
	meanDeltas := make([]float64, testcaseCacheABBootstrapSamples)
	memorySample, cachedSample := make([]float64, len(memory)), make([]float64, len(cached))
	for i := range medianDeltas {
		for j := range memorySample {
			memorySample[j], cachedSample[j] = memory[rng.Intn(len(memory))], cached[rng.Intn(len(cached))]
		}
		memoryMean, cachedMean := 0.0, 0.0
		for j := range memorySample {
			memoryMean += memorySample[j]
			cachedMean += cachedSample[j]
		}
		meanDeltas[i] = memoryMean/float64(len(memorySample)) - cachedMean/float64(len(cachedSample))
		sort.Float64s(memorySample)
		sort.Float64s(cachedSample)
		medianDeltas[i] = transportABQuantile(memorySample, .5) - transportABQuantile(cachedSample, .5)
	}
	sort.Float64s(medianDeltas)
	sort.Float64s(meanDeltas)
	memorySorted, cachedSorted := append([]float64(nil), memory...), append([]float64(nil), cached...)
	sort.Float64s(memorySorted)
	sort.Float64s(cachedSorted)
	return testcaseCacheABCI{Method: "nonparametric bootstrap of independently resampled mode statistics", Seed: testcaseCacheABBootstrapSeed, Resamples: testcaseCacheABBootstrapSamples, MedianDeltaMS: transportABQuantile(memorySorted, .5) - transportABQuantile(cachedSorted, .5), MedianLower95MS: transportABQuantile(medianDeltas, .025), MedianUpper95MS: transportABQuantile(medianDeltas, .975), MeanDeltaMS: mean(memory) - mean(cached), MeanLower95MS: transportABQuantile(meanDeltas, .025), MeanUpper95MS: transportABQuantile(meanDeltas, .975)}
}

func mean(values []float64) float64 {
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func testcaseCacheABAssessment(p50 testcaseCacheABDelta, ci testcaseCacheABCI) string {
	if ci.MedianLower95MS <= 0 && ci.MedianUpper95MS >= 0 {
		return "NO MATERIAL EFFECT (median CI crosses zero)"
	}
	switch {
	case p50.ImprovementPct >= 20:
		return "LARGE EFFECT (local p50 reduction >=20%, CI excludes zero)"
	case p50.ImprovementPct >= 10:
		return "MODERATE EFFECT (local p50 reduction >=10%, CI excludes zero)"
	case p50.ImprovementPct > 0:
		return "SMALL EFFECT (local p50 reduction <10%, CI excludes zero)"
	default:
		return "NO MATERIAL EFFECT (CachedFile was not faster at p50)"
	}
}

func testcaseCacheABReport(summary testcaseCacheABSummary) string {
	lines := []string{"# gRPC MemoryFile vs CachedFile — Judge Core", "", "Synthetic local C++ workload. Compile was performed once via the actual production GoJudgeClient and is excluded from every Judge Core sample. Each submission executes 14 accepted testcases in official batches of 4, 4, 4, 2 at concurrency 1.", "", "The only benchmark variable is testcase stdin representation: `MemoryFile(content)` versus the current production testcase-cache `CachedFile(FileID)` path. The same executable FileID was used by both modes in one sandbox process.", "", "## Hot-cache Judge Core latency", "", "| Input size | Mode | p50 | p95 | p99 | Mean |", "|---|---|---:|---:|---:|---:|"}
	for _, class := range summary.Classes {
		lines = append(lines, fmt.Sprintf("| %s (%s) | MemoryFile | %.3f ms | %.3f ms | %.3f ms | %.3f ms |", class.InputSizeClass, testcaseCacheABFormatBytes(class.BytesPerTestcase), class.MemoryFile.TotalTestcaseWall.P50, class.MemoryFile.TotalTestcaseWall.P95, class.MemoryFile.TotalTestcaseWall.P99, class.MemoryFile.TotalTestcaseWall.Mean), fmt.Sprintf("| %s (%s) | CachedFile | %.3f ms | %.3f ms | %.3f ms | %.3f ms |", class.InputSizeClass, testcaseCacheABFormatBytes(class.BytesPerTestcase), class.CachedFile.TotalTestcaseWall.P50, class.CachedFile.TotalTestcaseWall.P95, class.CachedFile.TotalTestcaseWall.P99, class.CachedFile.TotalTestcaseWall.Mean))
	}
	lines = append(lines, "", "| Input size | p50 gain | p95 gain | p99 gain | Median delta 95% CI |", "|---|---:|---:|---:|---:|")
	for _, class := range summary.Classes {
		lines = append(lines, fmt.Sprintf("| %s | %.3f ms (%.2f%%) | %.3f ms (%.2f%%) | %.3f ms (%.2f%%) | %.3f ms [%.3f, %.3f] ms |", testcaseCacheABFormatBytes(class.BytesPerTestcase), class.Delta["p50"].AbsoluteMS, class.Delta["p50"].ImprovementPct, class.Delta["p95"].AbsoluteMS, class.Delta["p95"].ImprovementPct, class.Delta["p99"].AbsoluteMS, class.Delta["p99"].ImprovementPct, class.MedianCI.MedianDeltaMS, class.MedianCI.MedianLower95MS, class.MedianCI.MedianUpper95MS))
	}
	lines = append(lines, "", "## Cold cache", "", "| Input size | FileAdd calls | Bytes uploaded | Population time | First cached execution | Cold end-to-end |", "|---|---:|---:|---:|---:|---:|")
	for _, cold := range summary.ColdCache {
		lines = append(lines, fmt.Sprintf("| %s | %d | %d | %.3f ms | %.3f ms | %.3f ms |", testcaseCacheABFormatBytes(cold.BytesPerTestcase), cold.FileAddCalls, cold.BytesUploaded, cold.PopulationDuration, cold.FirstExecutionWall, cold.ColdEndToEnd))
	}
	lines = append(lines, "", "## Data movement", "", "| Input size | MemoryFile bytes/submission | CachedFile hot bytes/submission | Eliminated testcase bytes |", "|---|---:|---:|---:|")
	for _, class := range summary.Classes {
		lines = append(lines, fmt.Sprintf("| %s | %d | 0 | %d |", testcaseCacheABFormatBytes(class.BytesPerTestcase), class.TotalInputBytes, class.TotalInputBytes))
	}
	lines = append(lines, "", "## Isolated service-rate estimate", "", "| Input size | MemoryFile | CachedFile |", "|---|---:|---:|")
	for _, class := range summary.Classes {
		lines = append(lines, fmt.Sprintf("| %s | %.3f sub/s | %.3f sub/s |", testcaseCacheABFormatBytes(class.BytesPerTestcase), class.MemoryFile.ServiceRate, class.CachedFile.ServiceRate))
	}
	lines = append(lines, "", "## Correctness and validity", "", fmt.Sprintf("All measured iterations were valid: 14 results, four expected batches, ACCEPTED verdicts, matching stdout, no malformed response, and no transport error. Hot CachedFile measurements had FileAdd count %v.", summary.Correctness["hot_cached_file_add_calls"]), "", "A median CI that crosses zero is classified as no material local effect. When it excludes zero, p50 reduction thresholds are explicitly descriptive: <10% small, 10–<20% moderate, and >=20% large. These local measurements are not a production-throughput claim.", "")
	for _, class := range summary.Classes {
		lines = append(lines, fmt.Sprintf("- %s: %s", testcaseCacheABFormatBytes(class.BytesPerTestcase), class.Assessment))
	}
	return strings.Join(lines, "\n") + "\n"
}

func testcaseCacheABFormatBytes(value int) string {
	if value%(1024*1024) == 0 {
		return fmt.Sprintf("%d MiB", value/(1024*1024))
	}
	return fmt.Sprintf("%d KiB", value/1024)
}

type testcaseCacheABResourceSample struct {
	TimestampUTC      time.Time
	CPUPct, MemoryMiB float64
	PIDs              int
	Error             string
}

type testcaseCacheABResourceSampler struct {
	container string
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.Mutex
	samples   []testcaseCacheABResourceSample
}

func newTestcaseCacheABResourceSampler(container string) *testcaseCacheABResourceSampler {
	return &testcaseCacheABResourceSampler{container: strings.TrimSpace(container)}
}
func (s *testcaseCacheABResourceSampler) Start(parent context.Context) {
	if s.container == "" {
		return
	}
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			s.capture(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
func (s *testcaseCacheABResourceSampler) Stop() {
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.wg.Wait()
}
func (s *testcaseCacheABResourceSampler) Samples() []testcaseCacheABResourceSample {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]testcaseCacheABResourceSample(nil), s.samples...)
}
func (s *testcaseCacheABResourceSampler) capture(ctx context.Context) {
	sample := testcaseCacheABResourceSample{TimestampUTC: time.Now().UTC()}
	command := exec.CommandContext(ctx, "docker", "stats", "--no-stream", "--format", "{{.CPUPerc}},{{.MemUsage}},{{.PIDs}}", s.container)
	output, err := command.Output()
	if err != nil && ctx.Err() != nil {
		// Stop intentionally cancels an in-flight diagnostic command. That is
		// not a failed resource sample and should not affect the benchmark.
		return
	}
	if err != nil {
		sample.Error = err.Error()
	} else {
		sample.CPUPct, sample.MemoryMiB, sample.PIDs, sample.Error = testcaseCacheABParseDockerStats(strings.TrimSpace(string(output)))
	}
	s.mu.Lock()
	s.samples = append(s.samples, sample)
	s.mu.Unlock()
}
func testcaseCacheABParseDockerStats(value string) (float64, float64, int, string) {
	parts := strings.Split(value, ",")
	if len(parts) != 3 {
		return 0, 0, 0, "unexpected docker stats format"
	}
	cpu, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(parts[0]), "%"), 64)
	if err != nil {
		return 0, 0, 0, err.Error()
	}
	memoryParts := strings.Fields(strings.TrimSpace(parts[1]))
	if len(memoryParts) < 1 {
		return 0, 0, 0, "missing docker memory usage"
	}
	memory, err := testcaseCacheABParseMemoryMiB(memoryParts[0])
	if err != nil {
		return 0, 0, 0, err.Error()
	}
	pids, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return 0, 0, 0, err.Error()
	}
	return cpu, memory, pids, ""
}
func testcaseCacheABParseMemoryMiB(value string) (float64, error) {
	units := []struct {
		suffix string
		factor float64
	}{{"GiB", 1024}, {"MiB", 1}, {"KiB", 1.0 / 1024}, {"B", 1.0 / (1024 * 1024)}}
	for _, unit := range units {
		if strings.HasSuffix(value, unit.suffix) {
			amount, err := strconv.ParseFloat(strings.TrimSuffix(value, unit.suffix), 64)
			return amount * unit.factor, err
		}
	}
	return 0, fmt.Errorf("unknown memory unit %q", value)
}
func testcaseCacheABWriteResourceCSV(path string, samples []testcaseCacheABResourceSample) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open resource_samples.csv: %w", err)
	}
	defer func() { _ = file.Close() }()
	writer := csv.NewWriter(file)
	if err := writer.Write([]string{"timestamp_utc", "cpu_percent", "memory_mib", "pids", "error"}); err != nil {
		return err
	}
	for _, sample := range samples {
		if err := writer.Write([]string{sample.TimestampUTC.Format(time.RFC3339Nano), strconv.FormatFloat(sample.CPUPct, 'f', 3, 64), strconv.FormatFloat(sample.MemoryMiB, 'f', 3, 64), strconv.Itoa(sample.PIDs), sample.Error}); err != nil {
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}
	return file.Sync()
}
func testcaseCacheABResourceSummary(samples []testcaseCacheABResourceSample) map[string]any {
	if len(samples) == 0 {
		return map[string]any{"available": false, "reason": "GO_JUDGE_TESTCASE_CACHE_AB_CONTAINER was not set"}
	}
	peakCPU, peakMemory, peakPIDs := 0.0, 0.0, 0
	failures := 0
	intervalTotal := time.Duration(0)
	intervalCount := 0
	var previous time.Time
	for _, sample := range samples {
		peakCPU = math.Max(peakCPU, sample.CPUPct)
		peakMemory = math.Max(peakMemory, sample.MemoryMiB)
		if sample.PIDs > peakPIDs {
			peakPIDs = sample.PIDs
		}
		if sample.Error != "" {
			failures++
		}
		if !previous.IsZero() && sample.TimestampUTC.After(previous) {
			intervalTotal += sample.TimestampUTC.Sub(previous)
			intervalCount++
		}
		previous = sample.TimestampUTC
	}
	observedInterval := 0.0
	if intervalCount > 0 {
		observedInterval = intervalTotal.Seconds() / float64(intervalCount)
	}
	return map[string]any{
		"available":                           true,
		"samples":                             len(samples),
		"requested_sampling_interval_seconds": 1,
		"observed_mean_sampling_interval_seconds": observedInterval,
		"peak_cpu_percent":                        peakCPU,
		"peak_memory_mib":                         peakMemory,
		"peak_pids":                               peakPIDs,
		"sampling_failures":                       failures,
	}
}

func testcaseCacheABRequiredEnv(t *testing.T, key string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		t.Skipf("set %s to run the local testcase-cache A/B integration benchmark", key)
	}
	return value
}
func testcaseCacheABResultsDir(t *testing.T) string {
	t.Helper()
	if value := strings.TrimSpace(os.Getenv("GO_JUDGE_TESTCASE_CACHE_AB_RESULTS_DIR")); value != "" {
		return value
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(cwd, "go.work")); err == nil {
			return filepath.Join(cwd, "tools", "benchmark", "judge", "bench-results", "testcase-cache-ab-"+time.Now().UTC().Format("20060102T150405Z"))
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			t.Fatal("locate repository root for benchmark results")
		}
		cwd = parent
	}
}
