package execute

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go-judge-system/pkg/gojudge"
	"go-judge-system/workers/judge/internal/application/port/outbound"
	workerdomain "go-judge-system/workers/judge/internal/domain"

	resty "github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

const (
	defaultBatchSize                  = 50
	officialBatchSize                 = 4
	expectedOutputHeadroomBytes       = 64 * 1024
	minCompileLimitMS           int64 = 30_000
)

type GoJudgeClient struct {
	client *resty.Client
	logger *zap.Logger
}

func NewGoJudgeClient(baseURL string, logger *zap.Logger) *GoJudgeClient {
	return &GoJudgeClient{
		client: resty.New().SetBaseURL(baseURL).SetTimeout(120 * time.Second),
		logger: logger,
	}
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

	for start := 0; start < len(req.TestCases); start += currentBatchSize {
		end := start + currentBatchSize
		if end > len(req.TestCases) {
			end = len(req.TestCases)
		}

		batch := req.TestCases[start:end]
		runResp, err := c.runBatch(ctx, req.Language, req.SourceCode, langCfg, limits, hasCompile, exeFileID, batch)
		if err != nil {
			return nil, err
		}
		if len(runResp) != len(batch) {
			return nil, fmt.Errorf("go-judge returned %d results for %d test cases", len(runResp), len(batch))
		}

		for i, raw := range runResp {
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

func (c *GoJudgeClient) compile(
	ctx context.Context,
	language string,
	sourceCode string,
	langCfg *gojudge.LanguageConfig,
	limits outbound.ExecutionLimits,
) (*outbound.ExecutionResult, string, error) {
	compileLimitMS := maxInt64(limits.TimeLimitMS, minCompileLimitMS)
	compileMemoryKB := maxInt64(limits.MemoryLimitKB, 512*1024)
	compileReq := gojudge.Request{
		Cmd: []*gojudge.Cmd{
			{
				Args: langCfg.Compile.Command,
				Env:  langCfg.Compile.Env,
				Files: []*gojudge.File{
					{Content: stringPtr("")},
					{Name: stringPtr("stdout"), Max: int64Ptr(limits.OutputLimitBytes)},
					{Name: stringPtr("stderr"), Max: int64Ptr(limits.OutputLimitBytes)},
				},
				CopyIn: map[string]*gojudge.File{
					gojudge.GetSourceFileName(language): {Content: &sourceCode},
				},
				CopyOut:       []string{"stdout", "stderr"},
				CopyOutCached: []string{gojudge.GetExeFileName(language)},
				MemoryLimit:   uint64(compileMemoryKB * 1024),
				CPULimit:      uint64(compileLimitMS * int64(time.Millisecond)),
				ClockLimit:    uint64(compileLimitMS * int64(time.Millisecond) * 2),
				ProcLimit:     500,
			},
		},
	}

	var compileResp gojudge.Response
	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(compileReq).
		SetResult(&compileResp).
		Post("/run")
	if err != nil {
		c.logger.Error("failed to call go-judge API for compilation", zap.Error(err))
		return nil, "", fmt.Errorf("call go-judge compile API: %w", err)
	}
	if resp.IsError() || len(compileResp) == 0 {
		return nil, "", fmt.Errorf("go-judge compile returned status: %d", resp.StatusCode())
	}
	c.logger.Debug("go-judge compile request completed")

	res := compileResp[0]
	if res.Status != "Accepted" {
		compileOutput := res.Error
		if f, ok := res.Files["stderr"]; ok && f != "" {
			compileOutput = f
		} else if f, ok := res.Files["stdout"]; ok && f != "" {
			compileOutput = f
		}
		compileOutput = sanitizeOutput(compileOutput)
		return &outbound.ExecutionResult{
			Status:        "COMPILATION_ERROR",
			CompileOutput: &compileOutput,
			Diagnostics:   parseCompileDiagnostics(language, compileOutput),
			TestCases:     []outbound.TestCaseResult{},
		}, "", nil
	}

	fileID, ok := res.FileIDs[gojudge.GetExeFileName(language)]
	if !ok {
		return nil, "", fmt.Errorf("compile succeeded but exe fileId not found in response")
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
	testCases []outbound.ExecutionTestCase,
) (gojudge.Response, error) {
	runReq := gojudge.Request{
		Cmd: make([]*gojudge.Cmd, 0, len(testCases)),
	}

	for _, testCase := range testCases {
		stdin := testCase.Stdin
		runCmd := &gojudge.Cmd{
			Args: langCfg.Run.Command,
			Env:  langCfg.Run.Env,
			Files: []*gojudge.File{
				{Content: &stdin},
				{Name: stringPtr("stdout"), Max: int64Ptr(limits.OutputLimitBytes)},
				{Name: stringPtr("stderr"), Max: int64Ptr(limits.OutputLimitBytes)},
			},
			CopyOut:     []string{"stdout", "stderr"},
			MemoryLimit: uint64(limits.MemoryLimitKB * 1024),
			CPULimit:    uint64(limits.TimeLimitMS * int64(time.Millisecond)),
			ClockLimit:  uint64(limits.TimeLimitMS * int64(time.Millisecond) * 2),
			ProcLimit:   50,
		}
		if hasCompile {
			runCmd.CopyIn = map[string]*gojudge.File{
				gojudge.GetExeFileName(language): {FileID: stringPtr(exeFileID)},
			}
		} else {
			runCmd.CopyIn = map[string]*gojudge.File{
				gojudge.GetSourceFileName(language): {Content: &sourceCode},
			}
		}
		runReq.Cmd = append(runReq.Cmd, runCmd)
	}

	var runResp gojudge.Response
	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(runReq).
		SetResult(&runResp).
		Post("/run")
	if err != nil {
		c.logger.Error("failed to call go-judge API", zap.Error(err))
		return nil, fmt.Errorf("call go-judge run API: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("go-judge run returned status: %d", resp.StatusCode())
	}
	c.logger.Debug("go-judge run request completed", zap.Int("testcases", len(testCases)))
	return runResp, nil
}

func mapTestCaseResult(
	language string,
	testCase outbound.ExecutionTestCase,
	res gojudge.Result,
	officialSubmission bool,
) outbound.TestCaseResult {
	status := mapJudgeStatus(res.Status, res.ExitStatus)
	if officialSubmission {
		status = mapOfficialSubmissionStatus(status)
	}
	stdout := res.Files["stdout"]
	stderr := sanitizeOutput(res.Files["stderr"])
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
		ExecutionTime:  int(res.Time / uint64(time.Millisecond)),
		MemoryUsed:     int(res.Memory / 1024),
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

func mapJudgeStatus(status string, exitStatus int) string {
	switch status {
	case "Accepted":
		if exitStatus != 0 {
			return "RUNTIME_ERROR"
		}
		return "ACCEPTED"
	case "Memory Limit Exceeded":
		return "MEMORY_LIMIT_EXCEEDED"
	case "Time Limit Exceeded":
		return "TIME_LIMIT_EXCEEDED"
	case "Output Limit Exceeded":
		return "OUTPUT_LIMIT_EXCEEDED"
	case "File Error", "Nonzero Exit Status", "Signalled", "Run Error", "Runtime Error":
		return "RUNTIME_ERROR"
	case "Internal Error":
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

func stringPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}
