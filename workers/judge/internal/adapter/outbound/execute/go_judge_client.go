package execute

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go-judge-system/pkg/config"
	"go-judge-system/pkg/gojudge"
	"go-judge-system/workers/judge/internal/application/port/outbound"
	workerdomain "go-judge-system/workers/judge/internal/domain"

	judgepb "github.com/criyle/go-judge/pb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	defaultBatchSize                  = 50
	officialBatchSize                 = 4
	expectedOutputHeadroomBytes       = 64 * 1024
	minCompileLimitMS           int64 = 30_000
	sandboxRPCTimeout                 = 120 * time.Second
	executableCleanupRPCTimeout       = 5 * time.Second
)

// executorRPC is deliberately limited to the unary operation used by the
// Worker. The production implementation is judgepb.ExecutorClient.
type executorRPC interface {
	Exec(context.Context, *judgepb.Request, ...grpc.CallOption) (*judgepb.Response, error)
	sandboxFileRPC
}

type GoJudgeClient struct {
	client        executorRPC
	logger        *zap.Logger
	testcaseCache *sandboxTestcaseCache
}

func NewGoJudgeClient(client executorRPC, logger *zap.Logger, testcaseCacheConfigs ...config.TestcaseCacheConfig) *GoJudgeClient {
	testcaseCacheCfg := config.TestcaseCacheConfig{}
	if len(testcaseCacheConfigs) > 0 {
		testcaseCacheCfg = testcaseCacheConfigs[0]
	}
	cache, err := newConfiguredSandboxTestcaseCache(client, logger, testcaseCacheCfg)
	if err != nil {
		// Production construction validates configuration before this point. Keep
		// direct callers safe: a disabled cache preserves MemoryFile judging.
		if logger != nil {
			logger.Warn("sandbox testcase cache disabled because its configuration is invalid", zap.Error(err))
		}
		cache, _ = newConfiguredSandboxTestcaseCache(client, logger, config.TestcaseCacheConfig{})
	}
	return &GoJudgeClient{client: client, logger: logger, testcaseCache: cache}
}

func (c *GoJudgeClient) Execute(ctx context.Context, req outbound.ExecutionRequest) (*outbound.ExecutionResult, error) {
	if strings.TrimSpace(req.Language) == "" {
		return nil, workerdomain.MarkNonRetryable(fmt.Errorf("language not specified"))
	}
	if req.SourceCode == "" {
		return nil, workerdomain.MarkNonRetryable(fmt.Errorf("source code is empty"))
	}
	if len(req.TestCases) == 0 {
		return nil, workerdomain.MarkNonRetryable(fmt.Errorf("no test cases provided"))
	}

	langCfg, ok := gojudge.GetLanguageConfig(req.Language, gojudge.GetSourceFileName(req.Language), gojudge.GetExeFileName(req.Language))
	if !ok {
		return nil, workerdomain.MarkNonRetryable(fmt.Errorf("unsupported language: %s", req.Language))
	}

	limits := normalizeLimits(req.Limits)
	limits = ensureOutputLimitForExpected(limits, req.TestCases)
	hasCompile := langCfg.Compile != nil
	var exeFileID string

	if hasCompile {
		compileResult, fileID, err := c.compile(ctx, req.Language, req.SourceCode, langCfg, limits)
		if err != nil {
			return nil, err
		}
		if compileResult != nil {
			return compileResult, nil
		}
		exeFileID = fileID
		// CopyOutCached creates a sandbox-owned executable that is only valid
		// for this submission. Keep it through every testcase batch, then
		// release it on every terminal path below. This is deliberately
		// separate from testcase-cache lifecycle management.
		defer c.cleanupExecutableFile(exeFileID)
	}

	result := &outbound.ExecutionResult{
		Status:        "ACCEPTED",
		TestCases:     make([]outbound.TestCaseResult, 0, len(req.TestCases)),
		ExecutionTime: 0,
		MemoryUsed:    0,
	}

	currentBatchSize := defaultBatchSize
	if req.StopOnFirstFailure {
		currentBatchSize = officialBatchSize
	}

	for start, batchIndex := 0, 0; start < len(req.TestCases); start, batchIndex = start+currentBatchSize, batchIndex+1 {
		end := start + currentBatchSize
		if end > len(req.TestCases) {
			end = len(req.TestCases)
		}

		batch := req.TestCases[start:end]
		runResp, err := c.runBatch(ctx, req.Language, req.SourceCode, langCfg, limits, hasCompile, exeFileID, req.TestcaseDataset, batch, batchIndex)
		if err != nil {
			return nil, err
		}
		if len(runResp) != len(batch) {
			return nil, fmt.Errorf("go-judge returned %d results for %d test cases", len(runResp), len(batch))
		}

		for i, raw := range runResp {
			if raw == nil {
				return nil, fmt.Errorf("go-judge returned an incomplete result for testcase %d", batch[i].Index)
			}
			tcResult := mapTestCaseResult(req.Language, batch[i], raw, req.StopOnFirstFailure)
			result.TestCases = append(result.TestCases, tcResult)
			if tcResult.ExecutionTime > result.ExecutionTime {
				result.ExecutionTime = tcResult.ExecutionTime
			}
			if tcResult.MemoryUsed > result.MemoryUsed {
				result.MemoryUsed = tcResult.MemoryUsed
			}

			if tcResult.Status != "ACCEPTED" && result.Status == "ACCEPTED" {
				result.Status = tcResult.Status
				result.ErrorMessage = errorMessageForTestCase(tcResult)
			}
			if req.StopOnFirstFailure && tcResult.Status != "ACCEPTED" {
				c.logger.Info(
					"early termination: non-ACCEPTED result detected, skipping remaining batches",
					zap.Int("tests_run", len(result.TestCases)),
					zap.Int("tests_total", len(req.TestCases)),
					zap.String("failing_status", tcResult.Status),
				)
				return result, nil
			}
		}
	}

	return result, nil
}

// cleanupExecutableFile is best-effort housekeeping for a submission-scoped
// CopyOutCached executable. It intentionally uses a small independent context:
// the job context may already be canceled when a batch fails, but cleanup must
// still have one bounded chance to release the sandbox file. A deletion failure
// must never replace the execution result or trigger a retry.
func (c *GoJudgeClient) cleanupExecutableFile(fileID string) {
	if fileID == "" || c == nil || c.client == nil {
		return
	}
	rpcCtx, cancel := context.WithTimeout(context.Background(), executableCleanupRPCTimeout)
	defer cancel()
	if _, err := c.client.FileDelete(rpcCtx, &judgepb.FileID{FileID: fileID}); err != nil {
		if c.logger != nil {
			c.logger.Warn("sandbox executable cleanup failed", zap.String("operation", "executable_cleanup"), zap.String("transport", "grpc"), zap.Error(err))
		}
	}
}

func (c *GoJudgeClient) compile(
	ctx context.Context,
	language string,
	sourceCode string,
	langCfg *gojudge.LanguageConfig,
	limits outbound.ExecutionLimits,
) (*outbound.ExecutionResult, string, error) {
	compileLimitMS := maxInt64(limits.TimeLimitMS, minCompileLimitMS)
	compileMemoryKB := maxInt64(limits.MemoryLimitKB, 512*1024)
	compileReq := &judgepb.Request{Cmd: []*judgepb.Request_CmdType{{
		Args: langCfg.Compile.Command,
		Env:  langCfg.Compile.Env,
		Files: []*judgepb.Request_File{
			memoryFile([]byte{}),
			pipeCollector("stdout", limits.OutputLimitBytes),
			pipeCollector("stderr", limits.OutputLimitBytes),
		},
		CopyIn: map[string]*judgepb.Request_File{
			gojudge.GetSourceFileName(language): memoryFile([]byte(sourceCode)),
		},
		CopyOut: []*judgepb.Request_CmdCopyOutFile{{Name: "stdout"}, {Name: "stderr"}},
		CopyOutCached: []*judgepb.Request_CmdCopyOutFile{{
			Name: gojudge.GetExeFileName(language),
		}},
		MemoryLimit:    uint64(compileMemoryKB * 1024),
		CpuTimeLimit:   uint64(compileLimitMS * int64(time.Millisecond)),
		ClockTimeLimit: uint64(compileLimitMS * int64(time.Millisecond) * 2),
		ProcLimit:      500,
	}}}

	compileResp, err := c.exec(ctx, "compile", 0, -1, compileReq)
	if err != nil {
		return nil, "", err
	}
	if len(compileResp.GetResults()) == 0 || compileResp.GetResults()[0] == nil {
		return nil, "", fmt.Errorf("go-judge compile response did not contain a result")
	}

	res := compileResp.GetResults()[0]
	if res.GetStatus() != judgepb.Response_Result_Accepted {
		compileOutput := res.GetError()
		if stderr := string(res.GetFiles()["stderr"]); stderr != "" {
			compileOutput = stderr
		} else if stdout := string(res.GetFiles()["stdout"]); stdout != "" {
			compileOutput = stdout
		}
		compileOutput = sanitizeOutput(compileOutput)
		return &outbound.ExecutionResult{
			Status:        "COMPILATION_ERROR",
			CompileOutput: &compileOutput,
			Diagnostics:   parseCompileDiagnostics(language, compileOutput),
			TestCases:     []outbound.TestCaseResult{},
		}, "", nil
	}

	fileID := res.GetFileIDs()[gojudge.GetExeFileName(language)]
	if fileID == "" {
		return nil, "", fmt.Errorf("compile succeeded but executable file ID was absent from response")
	}
	return nil, fileID, nil
}

func (c *GoJudgeClient) runBatch(
	ctx context.Context,
	language string,
	sourceCode string,
	langCfg *gojudge.LanguageConfig,
	limits outbound.ExecutionLimits,
	hasCompile bool,
	exeFileID string,
	testcaseDataset *outbound.TestcaseDatasetIdentity,
	testCases []outbound.ExecutionTestCase,
	batchIndex int,
) ([]*judgepb.Response_Result, error) {
	runReq, bindings, err := c.newRunBatchRequest(ctx, language, sourceCode, langCfg, limits, hasCompile, exeFileID, testcaseDataset, testCases)
	if err != nil {
		return nil, err
	}
	runResp, err := c.exec(ctx, "run_batch", len(testCases), batchIndex, runReq)
	c.testcaseCache.release(bindings)
	if err != nil {
		return nil, err
	}
	if hasFileErrors(runResp) && c.testcaseCache.invalidateMissing(ctx, bindings) {
		c.logger.Debug("sandbox testcase cache stale FileID recovered; retrying batch once", zap.String("operation", "testcase_cache"), zap.String("event", "stale"), zap.Int("batch_index", batchIndex))
		var retryBindings []testcaseCacheBinding
		runReq, retryBindings, err = c.newRunBatchRequest(ctx, language, sourceCode, langCfg, limits, hasCompile, exeFileID, testcaseDataset, testCases)
		if err != nil {
			return nil, err
		}
		runResp, err = c.exec(ctx, "run_batch", len(testCases), batchIndex, runReq)
		c.testcaseCache.release(retryBindings)
		if err != nil {
			return nil, err
		}
	}
	return runResp.GetResults(), nil
}

func (c *GoJudgeClient) newRunBatchRequest(
	ctx context.Context,
	language string,
	sourceCode string,
	langCfg *gojudge.LanguageConfig,
	limits outbound.ExecutionLimits,
	hasCompile bool,
	exeFileID string,
	testcaseDataset *outbound.TestcaseDatasetIdentity,
	testCases []outbound.ExecutionTestCase,
) (*judgepb.Request, []testcaseCacheBinding, error) {
	runReq := &judgepb.Request{Cmd: make([]*judgepb.Request_CmdType, 0, len(testCases))}
	bindings := make([]testcaseCacheBinding, 0, len(testCases))

	for _, testCase := range testCases {
		stdin := memoryFile([]byte(testCase.Stdin))
		if binding, cached, err := c.testcaseCache.getOrAdd(ctx, testcaseDataset, testCase); err != nil {
			c.testcaseCache.release(bindings)
			return nil, nil, err
		} else if cached {
			stdin = cachedFile(binding.fileID)
			bindings = append(bindings, binding)
		}
		runCmd := &judgepb.Request_CmdType{
			Args: langCfg.Run.Command,
			Env:  langCfg.Run.Env,
			Files: []*judgepb.Request_File{
				stdin,
				pipeCollector("stdout", limits.OutputLimitBytes),
				pipeCollector("stderr", limits.OutputLimitBytes),
			},
			CopyOut:        []*judgepb.Request_CmdCopyOutFile{{Name: "stdout"}, {Name: "stderr"}},
			MemoryLimit:    uint64(limits.MemoryLimitKB * 1024),
			CpuTimeLimit:   uint64(limits.TimeLimitMS * int64(time.Millisecond)),
			ClockTimeLimit: uint64(limits.TimeLimitMS * int64(time.Millisecond) * 2),
			ProcLimit:      50,
		}
		if hasCompile {
			runCmd.CopyIn = map[string]*judgepb.Request_File{
				gojudge.GetExeFileName(language): cachedFile(exeFileID),
			}
		} else {
			runCmd.CopyIn = map[string]*judgepb.Request_File{
				gojudge.GetSourceFileName(language): memoryFile([]byte(sourceCode)),
			}
		}
		runReq.Cmd = append(runReq.Cmd, runCmd)
	}
	return runReq, bindings, nil
}

func hasFileErrors(response *judgepb.Response) bool {
	for _, result := range response.GetResults() {
		if result != nil && len(result.GetFileError()) > 0 {
			return true
		}
	}
	return false
}

func (c *GoJudgeClient) exec(
	ctx context.Context,
	operation string,
	testcaseCount int,
	batchIndex int,
	req *judgepb.Request,
) (*judgepb.Response, error) {
	if c.client == nil {
		return nil, fmt.Errorf("go-judge gRPC client is not configured")
	}

	rpcCtx, cancel := context.WithTimeout(ctx, sandboxRPCTimeout)
	defer cancel()

	started := time.Now()
	resp, err := c.client.Exec(rpcCtx, req)
	fields := []zap.Field{
		zap.String("operation", operation),
		zap.String("transport", "grpc"),
		zap.Int("testcase_count", testcaseCount),
		zap.Int64("duration_ms", time.Since(started).Milliseconds()),
	}
	if batchIndex >= 0 {
		fields = append(fields, zap.Int("batch_index", batchIndex))
	}
	if err != nil {
		fields = append(fields, zap.String("status", "error"))
		c.logger.Debug("go-judge sandbox RPC failed", fields...)
		if ctxErr := rpcCtx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("go-judge %s RPC canceled or timed out: %w", operation, ctxErr)
		}
		return nil, fmt.Errorf("go-judge %s gRPC transport failure: %w", operation, err)
	}
	if resp == nil {
		fields = append(fields, zap.String("status", "incomplete_response"))
		c.logger.Debug("go-judge sandbox RPC completed", fields...)
		return nil, fmt.Errorf("go-judge %s RPC returned an empty response", operation)
	}
	if resp.GetError() != "" && len(resp.GetResults()) == 0 {
		fields = append(fields, zap.String("status", "response_error"))
		c.logger.Debug("go-judge sandbox RPC completed", fields...)
		return nil, fmt.Errorf("go-judge %s RPC returned an execution error", operation)
	}
	fields = append(fields, zap.String("status", "ok"))
	c.logger.Debug("go-judge sandbox RPC completed", fields...)
	return resp, nil
}

func memoryFile(content []byte) *judgepb.Request_File {
	return &judgepb.Request_File{File: &judgepb.Request_File_Memory{Memory: &judgepb.Request_MemoryFile{Content: content}}}
}

func cachedFile(fileID string) *judgepb.Request_File {
	return &judgepb.Request_File{File: &judgepb.Request_File_Cached{Cached: &judgepb.Request_CachedFile{FileID: fileID}}}
}

func pipeCollector(name string, max int64) *judgepb.Request_File {
	return &judgepb.Request_File{File: &judgepb.Request_File_Pipe{Pipe: &judgepb.Request_PipeCollector{Name: name, Max: max}}}
}

func mapTestCaseResult(
	language string,
	testCase outbound.ExecutionTestCase,
	res *judgepb.Response_Result,
	officialSubmission bool,
) outbound.TestCaseResult {
	status := mapJudgeStatus(res.GetStatus(), res.GetExitStatus())
	if officialSubmission {
		status = mapOfficialSubmissionStatus(status)
	}
	stdout := string(res.GetFiles()["stdout"])
	stderr := sanitizeOutput(string(res.GetFiles()["stderr"]))
	diagnostics := parseRuntimeDiagnostics(language, testCase.ID, stderr)

	if status == "ACCEPTED" && testCase.ExpectedOutput != nil && !workerdomain.OutputEqual(stdout, *testCase.ExpectedOutput) {
		status = "WRONG_ANSWER"
	}

	actual := stdout
	var actualOutput *string
	if actual != "" {
		actualOutput = &actual
	}
	var stderrOutput *string
	if stderr != "" {
		stderrOutput = &stderr
	}

	return outbound.TestCaseResult{
		Index:          testCase.Index,
		ID:             testCase.ID,
		Kind:           testCase.Kind,
		Status:         status,
		ActualOutput:   actualOutput,
		Stderr:         stderrOutput,
		ExpectedOutput: testCase.ExpectedOutput,
		ExecutionTime:  int(res.GetTime() / uint64(time.Millisecond)),
		MemoryUsed:     int(res.GetMemory() / 1024),
		Diagnostics:    diagnostics,
	}
}

func errorMessageForTestCase(testCase outbound.TestCaseResult) *string {
	var message string
	switch testCase.Status {
	case "RUNTIME_ERROR":
		stderr := ""
		if testCase.Stderr != nil {
			stderr = *testCase.Stderr
		}
		message = runtimeErrorMessage(stderr, testCase.Diagnostics)
	case "TIME_LIMIT_EXCEEDED":
		message = "Execution exceeded the time limit."
	case "MEMORY_LIMIT_EXCEEDED":
		message = "Execution exceeded the memory limit."
	case "SYSTEM_ERROR":
		message = "The judge could not complete this submission."
	}
	if strings.TrimSpace(message) == "" {
		return nil
	}
	return &message
}

func mapJudgeStatus(status judgepb.Response_Result_StatusType, exitStatus int32) string {
	switch status {
	case judgepb.Response_Result_Accepted:
		if exitStatus != 0 {
			return "RUNTIME_ERROR"
		}
		return "ACCEPTED"
	case judgepb.Response_Result_MemoryLimitExceeded:
		return "MEMORY_LIMIT_EXCEEDED"
	case judgepb.Response_Result_TimeLimitExceeded:
		return "TIME_LIMIT_EXCEEDED"
	case judgepb.Response_Result_OutputLimitExceeded:
		return "OUTPUT_LIMIT_EXCEEDED"
	case judgepb.Response_Result_FileError, judgepb.Response_Result_NonZeroExitStatus, judgepb.Response_Result_Signalled:
		return "RUNTIME_ERROR"
	case judgepb.Response_Result_InternalError:
		return "SYSTEM_ERROR"
	default:
		return "SYSTEM_ERROR"
	}
}

func mapOfficialSubmissionStatus(status string) string {
	return status
}

func normalizeLimits(limits outbound.ExecutionLimits) outbound.ExecutionLimits {
	if limits.TimeLimitMS <= 0 {
		limits.TimeLimitMS = 2_000
	}
	if limits.MemoryLimitKB <= 0 {
		limits.MemoryLimitKB = 256 * 1024
	}
	if limits.OutputLimitBytes <= 0 {
		limits.OutputLimitBytes = 1024 * 1024
	}
	return limits
}

func ensureOutputLimitForExpected(
	limits outbound.ExecutionLimits,
	testCases []outbound.ExecutionTestCase,
) outbound.ExecutionLimits {
	required := limits.OutputLimitBytes
	for _, testCase := range testCases {
		if testCase.ExpectedOutput == nil {
			continue
		}
		size := int64(len(*testCase.ExpectedOutput)) + expectedOutputHeadroomBytes
		if size > required {
			required = size
		}
	}
	limits.OutputLimitBytes = required
	return limits
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
