//go:build integration

package execute

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"go-judge-system/pkg/gojudge"
	sharedgrpc "go-judge-system/pkg/grpc"
	"go-judge-system/workers/judge/internal/application/port/outbound"
	workerdomain "go-judge-system/workers/judge/internal/domain"

	judgepb "github.com/criyle/go-judge/pb"
	"go.uber.org/zap"
)

const (
	transportABWarmupIterations = 10
	transportABMeasurements     = 100
	transportABBootstrapSamples = 10_000
	transportABBootstrapSeed    = int64(20_260_828)
)

// TestGoJudgeTransportAB is deliberately opt-in: it requires a local sandbox
// with both endpoints exposed and writes local ignored benchmark artifacts.
// It keeps the v0.1.9 HTTP /run behavior in test-only code; production uses
// only GoJudgeClient's gRPC implementation.
func TestGoJudgeTransportAB(t *testing.T) {
	grpcAddress := transportABRequiredEnv(t, "GO_JUDGE_TRANSPORT_AB_GRPC_ADDR")
	httpURL := strings.TrimRight(transportABRequiredEnv(t, "GO_JUDGE_TRANSPORT_AB_HTTP_URL"), "/")
	resultsDir := transportABRequiredEnv(t, "GO_JUDGE_TRANSPORT_AB_RESULTS_DIR")
	if err := os.MkdirAll(resultsDir, 0o700); err != nil {
		t.Fatalf("create result directory: %v", err)
	}

	conn, err := sharedgrpc.NewClientConn(grpcAddress, sharedgrpc.WithInsecureTransport())
	if err != nil {
		t.Fatalf("create go-judge gRPC connection: %v", err)
	}
	defer func() { _ = conn.Close() }()

	grpcClient := NewGoJudgeClient(judgepb.NewExecutorClient(conn), zap.NewNop())
	httpClient := &http.Client{Timeout: sandboxRPCTimeout}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	const source = `#include <bits/stdc++.h>
using namespace std;

int main() {
	long long a, b;
	if (!(cin >> a >> b)) return 1;
	volatile unsigned long long work = static_cast<unsigned long long>(a) ^ (static_cast<unsigned long long>(b) << 1);
	for (unsigned long long i = 0; i < 50000; ++i) {
		work = work * 6364136223846793005ULL + i + 1442695040888963407ULL;
	}
	cout << a + b << '\n';
	return work == 0xFFFFFFFFFFFFFFFFULL ? 1 : 0;
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
	testCases := transportABSyntheticTestCases()

	compileStarted := time.Now()
	compileResult, executableFileID, err := grpcClient.compile(ctx, "CPP", source, langCfg, limits)
	compileDuration := time.Since(compileStarted)
	if err != nil {
		t.Fatalf("compile through gRPC: %v", err)
	}
	if compileResult != nil {
		t.Fatalf("compile result = %#v, want successful compile", compileResult)
	}
	if executableFileID == "" {
		t.Fatal("compile returned an empty executable FileID")
	}

	for i := 0; i < transportABWarmupIterations; i++ {
		if _, err := transportABRunHTTP(ctx, httpClient, httpURL, source, langCfg, limits, executableFileID, testCases); err != nil {
			t.Fatalf("HTTP warmup %d: %v", i+1, err)
		}
	}
	for i := 0; i < transportABWarmupIterations; i++ {
		if _, err := transportABRunGRPC(ctx, grpcClient, source, langCfg, limits, executableFileID, testCases, -i-1); err != nil {
			t.Fatalf("gRPC warmup %d: %v", i+1, err)
		}
	}

	order := make([]string, 0, transportABMeasurements*2)
	for i := 0; i < transportABMeasurements; i++ {
		order = append(order, "http", "grpc")
	}
	rng := rand.New(rand.NewSource(transportABBootstrapSeed))
	rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })

	records := make([]transportABRecord, 0, len(order))
	nextIteration := map[string]int{"http": 0, "grpc": 0}
	for _, transport := range order {
		nextIteration[transport]++
		var record transportABRecord
		if transport == "http" {
			record, err = transportABRunHTTP(ctx, httpClient, httpURL, source, langCfg, limits, executableFileID, testCases)
		} else {
			record, err = transportABRunGRPC(ctx, grpcClient, source, langCfg, limits, executableFileID, testCases, nextIteration[transport]-1)
		}
		record.Transport = transport
		record.Iteration = nextIteration[transport]
		record.TimestampUTC = time.Now().UTC()
		if err != nil {
			record.TransportError = err.Error()
			records = append(records, record)
			_ = transportABWriteArtifacts(resultsDir, compileDuration, records)
			t.Fatalf("%s measurement %d correctness/transport failure: %v", transport, record.Iteration, err)
		}
		records = append(records, record)
	}

	if err := transportABWriteArtifacts(resultsDir, compileDuration, records); err != nil {
		t.Fatalf("write benchmark artifacts: %v", err)
	}
	t.Logf("HTTP vs gRPC benchmark complete: %s", resultsDir)
}

type transportABRecord struct {
	TimestampUTC       time.Time
	Transport          string
	Iteration          int
	TotalWall          time.Duration
	BatchWall          []time.Duration
	BatchCommandCounts []int
	BatchResultCounts  []int
	FinalVerdict       string
	TransportError     string
	MalformedResponse  bool
}

func transportABRunGRPC(
	ctx context.Context,
	client *GoJudgeClient,
	source string,
	langCfg *gojudge.LanguageConfig,
	limits outbound.ExecutionLimits,
	executableFileID string,
	testCases []outbound.ExecutionTestCase,
	batchIndexBase int,
) (transportABRecord, error) {
	started := time.Now()
	record := transportABRecord{FinalVerdict: "ACCEPTED"}
	for start, batchIndex := 0, 0; start < len(testCases); start, batchIndex = start+officialBatchSize, batchIndex+1 {
		end := min(start+officialBatchSize, len(testCases))
		batch := testCases[start:end]
		batchStarted := time.Now()
		results, err := client.runBatch(ctx, "CPP", source, langCfg, limits, true, executableFileID, nil, batch, batchIndexBase*10+batchIndex)
		record.BatchWall = append(record.BatchWall, time.Since(batchStarted))
		record.BatchCommandCounts = append(record.BatchCommandCounts, len(batch))
		record.BatchResultCounts = append(record.BatchResultCounts, len(results))
		if err != nil {
			return record, err
		}
		if err := transportABValidateGRPCResults(batch, results); err != nil {
			record.MalformedResponse = len(results) != len(batch)
			return record, err
		}
	}
	record.TotalWall = time.Since(started)
	return record, nil
}

func transportABRunHTTP(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	source string,
	langCfg *gojudge.LanguageConfig,
	limits outbound.ExecutionLimits,
	executableFileID string,
	testCases []outbound.ExecutionTestCase,
) (transportABRecord, error) {
	started := time.Now()
	record := transportABRecord{FinalVerdict: "ACCEPTED"}
	for start := 0; start < len(testCases); start += officialBatchSize {
		end := min(start+officialBatchSize, len(testCases))
		batch := testCases[start:end]
		batchStarted := time.Now()
		results, err := transportABHTTPRunBatch(ctx, client, baseURL, source, langCfg, limits, executableFileID, batch)
		record.BatchWall = append(record.BatchWall, time.Since(batchStarted))
		record.BatchCommandCounts = append(record.BatchCommandCounts, len(batch))
		record.BatchResultCounts = append(record.BatchResultCounts, len(results))
		if err != nil {
			return record, err
		}
		if err := transportABValidateHTTPResults(batch, results); err != nil {
			record.MalformedResponse = len(results) != len(batch)
			return record, err
		}
	}
	record.TotalWall = time.Since(started)
	return record, nil
}

// transportABHTTPRunBatch reconstructs only the v0.1.9 /run request used by
// the former Worker adapter. It is benchmark-only and intentionally does not
// reintroduce HTTP into the production dependency-injection path.
func transportABHTTPRunBatch(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	source string,
	langCfg *gojudge.LanguageConfig,
	limits outbound.ExecutionLimits,
	executableFileID string,
	testCases []outbound.ExecutionTestCase,
) (gojudge.Response, error) {
	reqPayload := gojudge.Request{Cmd: make([]*gojudge.Cmd, 0, len(testCases))}
	for _, testCase := range testCases {
		stdin := testCase.Stdin
		reqPayload.Cmd = append(reqPayload.Cmd, &gojudge.Cmd{
			Args: langCfg.Run.Command,
			Env:  langCfg.Run.Env,
			Files: []*gojudge.File{
				{Content: &stdin},
				{Name: transportABStringPtr("stdout"), Max: transportABInt64Ptr(limits.OutputLimitBytes)},
				{Name: transportABStringPtr("stderr"), Max: transportABInt64Ptr(limits.OutputLimitBytes)},
			},
			CopyIn: map[string]*gojudge.File{
				gojudge.GetExeFileName("CPP"): {FileID: transportABStringPtr(executableFileID)},
			},
			CopyOut:     []string{"stdout", "stderr"},
			MemoryLimit: uint64(limits.MemoryLimitKB * 1024),
			CPULimit:    uint64(limits.TimeLimitMS * int64(time.Millisecond)),
			ClockLimit:  uint64(limits.TimeLimitMS * int64(time.Millisecond) * 2),
			ProcLimit:   50,
		})
	}
	body, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal v0.1.9 HTTP /run request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/run", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("construct HTTP /run request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP /run transport: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("HTTP /run status: %s", resp.Status)
	}
	var result gojudge.Response
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 16*1024*1024))
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("decode HTTP /run response: %w", err)
	}
	return result, nil
}

func transportABValidateGRPCResults(testCases []outbound.ExecutionTestCase, results []*judgepb.Response_Result) error {
	if len(results) != len(testCases) {
		return fmt.Errorf("gRPC returned %d results for %d testcase commands", len(results), len(testCases))
	}
	for i, result := range results {
		if result == nil {
			return fmt.Errorf("gRPC returned a nil result for testcase %d", testCases[i].Index)
		}
		mapped := mapTestCaseResult("CPP", testCases[i], result, true)
		if mapped.Status != "ACCEPTED" {
			return fmt.Errorf("gRPC testcase %d verdict = %s", testCases[i].Index, mapped.Status)
		}
		if mapped.ActualOutput == nil || !workerdomain.OutputEqual(*mapped.ActualOutput, *testCases[i].ExpectedOutput) {
			return fmt.Errorf("gRPC testcase %d stdout did not match expected output", testCases[i].Index)
		}
	}
	return nil
}

func transportABValidateHTTPResults(testCases []outbound.ExecutionTestCase, results gojudge.Response) error {
	if len(results) != len(testCases) {
		return fmt.Errorf("HTTP returned %d results for %d testcase commands", len(results), len(testCases))
	}
	for i, result := range results {
		if result.Status != "Accepted" || result.ExitStatus != 0 {
			return fmt.Errorf("HTTP testcase %d status = %q exit_status = %d", testCases[i].Index, result.Status, result.ExitStatus)
		}
		if !workerdomain.OutputEqual(result.Files["stdout"], *testCases[i].ExpectedOutput) {
			return fmt.Errorf("HTTP testcase %d stdout did not match expected output", testCases[i].Index)
		}
	}
	return nil
}

func transportABWriteArtifacts(resultsDir string, compileDuration time.Duration, records []transportABRecord) error {
	if err := transportABWriteRawCSV(filepath.Join(resultsDir, "raw.csv"), records); err != nil {
		return err
	}
	summary, err := transportABSummary(compileDuration, records)
	if err != nil {
		return err
	}
	jsonBody, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal summary: %w", err)
	}
	if err := os.WriteFile(filepath.Join(resultsDir, "summary.json"), append(jsonBody, '\n'), 0o600); err != nil {
		return fmt.Errorf("write summary.json: %w", err)
	}
	if err := os.WriteFile(filepath.Join(resultsDir, "report.md"), []byte(transportABReport(summary)), 0o600); err != nil {
		return fmt.Errorf("write report.md: %w", err)
	}
	return nil
}

func transportABWriteRawCSV(path string, records []transportABRecord) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open raw.csv: %w", err)
	}
	defer func() { _ = file.Close() }()
	writer := csv.NewWriter(file)
	if err := writer.Write([]string{"timestamp_utc", "transport", "iteration", "total_testcase_wall_ms", "batch_wall_ms", "batch_command_counts", "batch_result_counts", "final_verdict", "transport_error", "malformed_response"}); err != nil {
		return err
	}
	for _, record := range records {
		if err := writer.Write([]string{
			record.TimestampUTC.Format(time.RFC3339Nano), record.Transport, strconv.Itoa(record.Iteration),
			transportABFormatMS(record.TotalWall), transportABDurationList(record.BatchWall), transportABIntList(record.BatchCommandCounts), transportABIntList(record.BatchResultCounts),
			record.FinalVerdict, record.TransportError, strconv.FormatBool(record.MalformedResponse),
		}); err != nil {
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("write raw.csv: %w", err)
	}
	return file.Sync()
}

type transportABStats struct {
	Count int     `json:"count"`
	Min   float64 `json:"min_ms"`
	Mean  float64 `json:"mean_ms"`
	P50   float64 `json:"p50_ms"`
	P90   float64 `json:"p90_ms"`
	P95   float64 `json:"p95_ms"`
	P99   float64 `json:"p99_ms"`
	Max   float64 `json:"max_ms"`
	Std   float64 `json:"std_ms"`
	CV    float64 `json:"cv"`
}

type transportABSummaryDocument struct {
	SchemaVersion int                         `json:"schema_version"`
	GeneratedAt   time.Time                   `json:"generated_at_utc"`
	Benchmark     string                      `json:"benchmark"`
	Workload      map[string]any              `json:"workload"`
	Compile       map[string]any              `json:"compile_diagnostic_excluded_from_kpi"`
	Warmup        map[string]int              `json:"warmup_iterations"`
	Measurements  map[string]int              `json:"measurement_iterations"`
	HTTP          transportABTransportSummary `json:"http"`
	GRPC          transportABTransportSummary `json:"grpc"`
	Delta         map[string]transportABDelta `json:"http_minus_grpc"`
	MedianCI      transportABMedianCI         `json:"median_delta_bootstrap_95_ci"`
	Correctness   map[string]any              `json:"correctness"`
}

type transportABTransportSummary struct {
	TotalTestcaseWall transportABStats            `json:"total_testcase_wall"`
	PerBatchWall      map[string]transportABStats `json:"per_batch_wall"`
	ServiceRate       float64                     `json:"isolated_service_rate_estimate_submissions_per_second"`
}

type transportABDelta struct {
	HTTPMS         float64 `json:"http_ms"`
	GRPCMS         float64 `json:"grpc_ms"`
	AbsoluteMS     float64 `json:"absolute_delta_ms"`
	ImprovementPct float64 `json:"improvement_pct"`
}

type transportABMedianCI struct {
	Method      string  `json:"method"`
	Seed        int64   `json:"seed"`
	Resamples   int     `json:"resamples"`
	MedianDelta float64 `json:"median_delta_ms"`
	Lower95     float64 `json:"lower_95_ms"`
	Upper95     float64 `json:"upper_95_ms"`
}

func transportABSummary(compileDuration time.Duration, records []transportABRecord) (transportABSummaryDocument, error) {
	byTransport := map[string][]transportABRecord{"http": nil, "grpc": nil}
	for _, record := range records {
		if record.TransportError != "" || record.MalformedResponse || record.FinalVerdict != "ACCEPTED" {
			return transportABSummaryDocument{}, fmt.Errorf("cannot summarize invalid %s measurement %d", record.Transport, record.Iteration)
		}
		byTransport[record.Transport] = append(byTransport[record.Transport], record)
	}
	if len(byTransport["http"]) != transportABMeasurements || len(byTransport["grpc"]) != transportABMeasurements {
		return transportABSummaryDocument{}, fmt.Errorf("measurement count http=%d grpc=%d, want %d each", len(byTransport["http"]), len(byTransport["grpc"]), transportABMeasurements)
	}
	httpSummary := transportABTransportStats(byTransport["http"])
	grpcSummary := transportABTransportStats(byTransport["grpc"])
	httpValues := transportABTotals(byTransport["http"])
	grpcValues := transportABTotals(byTransport["grpc"])
	ci := transportABBootstrapMedianCI(httpValues, grpcValues)
	return transportABSummaryDocument{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC(),
		Benchmark:     "HTTP REST /run vs gRPC pb.Executor.Exec Judge Core transport A/B",
		Workload: map[string]any{
			"language": "CPP", "synthetic": true, "testcase_count": 14,
			"official_batch_size": officialBatchSize, "batch_command_counts": []int{4, 4, 4, 2},
			"concurrency": 1, "compiled_executable_reused_across_transports": true,
		},
		Compile:      map[string]any{"transport": "grpc", "duration_ms": float64(compileDuration) / float64(time.Millisecond), "included_in_judge_core_kpi": false},
		Warmup:       map[string]int{"http": transportABWarmupIterations, "grpc": transportABWarmupIterations},
		Measurements: map[string]int{"http": len(byTransport["http"]), "grpc": len(byTransport["grpc"])},
		HTTP:         httpSummary,
		GRPC:         grpcSummary,
		Delta: map[string]transportABDelta{
			"p50":  transportABMakeDelta(httpSummary.TotalTestcaseWall.P50, grpcSummary.TotalTestcaseWall.P50),
			"p95":  transportABMakeDelta(httpSummary.TotalTestcaseWall.P95, grpcSummary.TotalTestcaseWall.P95),
			"p99":  transportABMakeDelta(httpSummary.TotalTestcaseWall.P99, grpcSummary.TotalTestcaseWall.P99),
			"mean": transportABMakeDelta(httpSummary.TotalTestcaseWall.Mean, grpcSummary.TotalTestcaseWall.Mean),
		},
		MedianCI:    ci,
		Correctness: map[string]any{"all_measured_iterations_valid": true, "correctness_failures": 0, "valid_http": len(byTransport["http"]), "valid_grpc": len(byTransport["grpc"])},
	}, nil
}

func transportABTransportStats(records []transportABRecord) transportABTransportSummary {
	batchValues := map[string][]float64{}
	for _, record := range records {
		for index, duration := range record.BatchWall {
			key := fmt.Sprintf("batch_%d", index+1)
			batchValues[key] = append(batchValues[key], float64(duration)/float64(time.Millisecond))
		}
	}
	perBatch := make(map[string]transportABStats, len(batchValues))
	for key, values := range batchValues {
		perBatch[key] = transportABCalculateStats(values)
	}
	total := transportABCalculateStats(transportABTotals(records))
	rate := 0.0
	if total.P50 > 0 {
		rate = 1000 / total.P50
	}
	return transportABTransportSummary{TotalTestcaseWall: total, PerBatchWall: perBatch, ServiceRate: rate}
}

func transportABTotals(records []transportABRecord) []float64 {
	values := make([]float64, 0, len(records))
	for _, record := range records {
		values = append(values, float64(record.TotalWall)/float64(time.Millisecond))
	}
	return values
}

func transportABCalculateStats(values []float64) transportABStats {
	if len(values) == 0 {
		return transportABStats{}
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mean := 0.0
	for _, value := range sorted {
		mean += value
	}
	mean /= float64(len(sorted))
	variance := 0.0
	for _, value := range sorted {
		variance += (value - mean) * (value - mean)
	}
	variance /= float64(len(sorted))
	std := math.Sqrt(variance)
	stats := transportABStats{Count: len(sorted), Min: sorted[0], Mean: mean, P50: transportABQuantile(sorted, .50), P90: transportABQuantile(sorted, .90), P95: transportABQuantile(sorted, .95), P99: transportABQuantile(sorted, .99), Max: sorted[len(sorted)-1], Std: std}
	if mean != 0 {
		stats.CV = std / mean
	}
	return stats
}

func transportABQuantile(sorted []float64, probability float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := probability * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	if lower == upper {
		return sorted[lower]
	}
	return sorted[lower] + (sorted[upper]-sorted[lower])*(index-float64(lower))
}

func transportABBootstrapMedianCI(httpValues, grpcValues []float64) transportABMedianCI {
	rng := rand.New(rand.NewSource(transportABBootstrapSeed))
	deltas := make([]float64, transportABBootstrapSamples)
	httpSample := make([]float64, len(httpValues))
	grpcSample := make([]float64, len(grpcValues))
	for i := range deltas {
		for sampleIndex := range httpSample {
			httpSample[sampleIndex] = httpValues[rng.Intn(len(httpValues))]
			grpcSample[sampleIndex] = grpcValues[rng.Intn(len(grpcValues))]
		}
		sort.Float64s(httpSample)
		sort.Float64s(grpcSample)
		deltas[i] = transportABQuantile(httpSample, .50) - transportABQuantile(grpcSample, .50)
	}
	sort.Float64s(deltas)
	httpSorted := append([]float64(nil), httpValues...)
	grpcSorted := append([]float64(nil), grpcValues...)
	sort.Float64s(httpSorted)
	sort.Float64s(grpcSorted)
	return transportABMedianCI{Method: "nonparametric bootstrap of independently resampled transport medians", Seed: transportABBootstrapSeed, Resamples: transportABBootstrapSamples, MedianDelta: transportABQuantile(httpSorted, .5) - transportABQuantile(grpcSorted, .5), Lower95: transportABQuantile(deltas, .025), Upper95: transportABQuantile(deltas, .975)}
}

func transportABMakeDelta(httpMS, grpcMS float64) transportABDelta {
	delta := transportABDelta{HTTPMS: httpMS, GRPCMS: grpcMS, AbsoluteMS: httpMS - grpcMS}
	if httpMS != 0 {
		delta.ImprovementPct = delta.AbsoluteMS / httpMS * 100
	}
	return delta
}

func transportABReport(summary transportABSummaryDocument) string {
	lines := []string{
		"# HTTP vs gRPC — Judge Core",
		"",
		"Synthetic local C++ workload: 14 accepted testcases in official batches of 4, 4, 4, 2; concurrency 1. The same gRPC-compiled executable FileID was reused by both transports in the same sandbox process.",
		"",
		"Compile excluded from Judge Core KPI. Compile was performed once via gRPC and recorded only as a diagnostic.",
		"",
		"| Metric | HTTP | gRPC | Delta |",
		"|---|---:|---:|---:|",
		transportABReportRow("p50 testcase wall", summary.Delta["p50"]),
		transportABReportRow("p95 testcase wall", summary.Delta["p95"]),
		transportABReportRow("p99 testcase wall", summary.Delta["p99"]),
		transportABReportRow("mean testcase wall", summary.Delta["mean"]),
		fmt.Sprintf("| isolated service-rate estimate | %.3f sub/s | %.3f sub/s | — |", summary.HTTP.ServiceRate, summary.GRPC.ServiceRate),
		"",
		fmt.Sprintf("Median delta bootstrap 95%% CI (HTTP − gRPC): %.3f ms [%.3f, %.3f] ms; %d deterministic resamples (seed %d).", summary.MedianCI.MedianDelta, summary.MedianCI.Lower95, summary.MedianCI.Upper95, summary.MedianCI.Resamples, summary.MedianCI.Seed),
		"",
		fmt.Sprintf("Compile diagnostic (excluded): gRPC %.3f ms.", summary.Compile["duration_ms"].(float64)),
		"",
		"## Per-batch wall time",
		"",
		"| Batch | HTTP p50/p95/p99 (ms) | gRPC p50/p95/p99 (ms) |",
		"|---|---:|---:|",
	}
	for _, batch := range []string{"batch_1", "batch_2", "batch_3", "batch_4"} {
		httpStats := summary.HTTP.PerBatchWall[batch]
		grpcStats := summary.GRPC.PerBatchWall[batch]
		lines = append(lines, fmt.Sprintf("| %s | %.3f / %.3f / %.3f | %.3f / %.3f / %.3f |", batch, httpStats.P50, httpStats.P95, httpStats.P99, grpcStats.P50, grpcStats.P95, grpcStats.P99))
	}
	lines = append(lines,
		"",
		"## Correctness gate",
		"",
		fmt.Sprintf("All %d HTTP and %d gRPC measured iterations returned all expected results, all ACCEPTED verdicts, and expected stdout. No measurement was silently discarded.", summary.Measurements["http"], summary.Measurements["grpc"]),
		"",
		"This isolated service-rate estimate is not a claim of sustainable production throughput. It excludes submission API, Kafka, queue wait, Problem Service, MinIO/testcase download, and compilation.",
		"",
	)
	return strings.Join(lines, "\n")
}

func transportABReportRow(label string, delta transportABDelta) string {
	return fmt.Sprintf("| %s | %.3f ms | %.3f ms | %.3f ms (%.2f%%) |", label, delta.HTTPMS, delta.GRPCMS, delta.AbsoluteMS, delta.ImprovementPct)
}

func transportABSyntheticTestCases() []outbound.ExecutionTestCase {
	inputs := [][3]string{
		{"1 2\n", "3\n", "one"}, {"100 200\n", "300\n", "two"}, {"-5 10\n", "5\n", "three"}, {"0 0\n", "0\n", "four"},
		{"999999 -1\n", "999998\n", "five"}, {"-100 -200\n", "-300\n", "six"}, {"42 58\n", "100\n", "seven"}, {"123456789 987654321\n", "1111111110\n", "eight"},
		{"-999999999 999999999\n", "0\n", "nine"}, {"7 -12\n", "-5\n", "ten"}, {"314159 271828\n", "585987\n", "eleven"}, {"-4000000000 1\n", "-3999999999\n", "twelve"},
		{"9223372036854775000 -10\n", "9223372036854774990\n", "thirteen"}, {"-9223372036854775000 10\n", "-9223372036854774990\n", "fourteen"},
	}
	testCases := make([]outbound.ExecutionTestCase, 0, len(inputs))
	for i, input := range inputs {
		expected := input[1]
		testCases = append(testCases, outbound.ExecutionTestCase{Index: i + 1, ID: input[2], Stdin: input[0], ExpectedOutput: &expected})
	}
	return testCases
}

func transportABRequiredEnv(t *testing.T, key string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		t.Skipf("set %s to run the local transport A/B integration benchmark", key)
	}
	return value
}

func transportABStringPtr(value string) *string { return &value }
func transportABInt64Ptr(value int64) *int64    { return &value }

func transportABFormatMS(duration time.Duration) string {
	return fmt.Sprintf("%.6f", float64(duration)/float64(time.Millisecond))
}

func transportABDurationList(values []time.Duration) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = transportABFormatMS(value)
	}
	return strings.Join(parts, ";")
}

func transportABIntList(values []int) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = strconv.Itoa(value)
	}
	return strings.Join(parts, ";")
}
