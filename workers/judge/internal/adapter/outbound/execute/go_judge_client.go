package execute

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-judge-system/pkg/gojudge"
	"go-judge-system/workers/judge/internal/application/port/outbound"

	resty "github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

// GoJudgeClient executes code using go-judge service.
// Uses shared volume for testcase input (Src field) instead of Content (in-memory).
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

func (c *GoJudgeClient) Execute(ctx context.Context, language, sourceCode string, bundle *outbound.TestCaseBundle) (*outbound.ExecutionResult, error) {
	if language == "" {
		return nil, fmt.Errorf("language not specified")
	}
	if sourceCode == "" {
		return nil, fmt.Errorf("source code is empty")
	}
	if bundle == nil || bundle.TestCount == 0 {
		return nil, fmt.Errorf("no test cases provided")
	}

	langCfg, ok := gojudge.GetLanguageConfig(language, gojudge.GetSourceFileName(language), gojudge.GetExeFileName(language))
	if !ok {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	const compileMemLimit = 512 * 1024 * 1024 // 512MB
	const runMemLimit = 256 * 1024 * 1024     // 256MB
	const compileTimeLimit = 15 * 1000000000  // 15s
	const runTimeLimit = 2 * 1000000000       // 2s

	hasCompile := langCfg.Compile != nil
	var exeFileID string

	// Step 1: Compilation (if needed)
	if hasCompile {
		compileReq := gojudge.Request{
			Cmd: []*gojudge.Cmd{
				{
					Args: langCfg.Compile.Command,
					Env:  langCfg.Compile.Env,
					Files: []*gojudge.File{
						{Content: stringPtr("")},                            // stdin
						{Name: stringPtr("stdout"), Max: int64Ptr(1048576)}, // stdout
						{Name: stringPtr("stderr"), Max: int64Ptr(1048576)}, // stderr
					},
					CopyIn: map[string]*gojudge.File{
						gojudge.GetSourceFileName(language): {Content: &sourceCode},
					},
					CopyOut:       []string{"stdout", "stderr"},
					CopyOutCached: []string{gojudge.GetExeFileName(language)},
					MemoryLimit:   compileMemLimit,
					CPULimit:      compileTimeLimit,
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
			return nil, fmt.Errorf("call go-judge compile API: %w", err)
		}

		if resp.IsError() || len(compileResp) == 0 {
			return nil, fmt.Errorf("go-judge compile returned status: %d", resp.StatusCode())
		}

		c.logger.Info("go-judge compile raw status", zap.Any("resp", compileResp))

		res := compileResp[0]
		if res.Status != "Accepted" {
			errStr := res.Error
			compileOutput := &errStr
			if f, ok := res.Files["stderr"]; ok && f != "" {
				compileOutput = &f
			} else if f, ok := res.Files["stdout"]; ok && f != "" {
				compileOutput = &f
			}
			return &outbound.ExecutionResult{
				Status:        "COMPILATION_ERROR",
				CompileOutput: compileOutput,
				TestCases:     []outbound.TestCaseResult{},
			}, nil
		}

		fileID, ok := res.FileIDs[gojudge.GetExeFileName(language)]
		if !ok {
			return nil, fmt.Errorf("compile succeeded but exe fileId not found in response")
		}
		exeFileID = fileID
	}

	// Step 2: Run test cases in batches to avoid overwhelming go-judge
	const batchSize = 50
	allResponses := make(gojudge.Response, 0, bundle.TestCount)

	for batchStart := 1; batchStart <= bundle.TestCount; batchStart += batchSize {
		batchEnd := batchStart + batchSize - 1
		if batchEnd > bundle.TestCount {
			batchEnd = bundle.TestCount
		}

		runReq := gojudge.Request{
			Cmd: make([]*gojudge.Cmd, 0, batchEnd-batchStart+1),
		}

		for i := batchStart; i <= batchEnd; i++ {
			inputPath := filepath.Join(bundle.Dir, fmt.Sprintf("%d.in", i))

			runCmd := &gojudge.Cmd{
				Args: langCfg.Run.Command,
				Env:  langCfg.Run.Env,
				Files: []*gojudge.File{
					{Src: stringPtr(inputPath)},
					{Name: stringPtr("stdout"), Max: int64Ptr(2 * 1024 * 1024)},
					{Name: stringPtr("stderr"), Max: int64Ptr(2 * 1024 * 1024)},
				},
				CopyOut:     []string{"stdout"},
				MemoryLimit: runMemLimit,
				CPULimit:    runTimeLimit,
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
			c.logger.Error("failed to call go-judge API",
				zap.Int("batch_start", batchStart),
				zap.Int("batch_end", batchEnd),
				zap.Error(err),
			)
			return nil, fmt.Errorf("call go-judge run API (batch %d-%d): %w", batchStart, batchEnd, err)
		}

		expectedLen := batchEnd - batchStart + 1
		if resp.IsError() || len(runResp) != expectedLen {
			return nil, fmt.Errorf("go-judge run status %d, expected %d results got %d",
				resp.StatusCode(), expectedLen, len(runResp))
		}

		allResponses = append(allResponses, runResp...)

		// Early termination: if any test in this batch failed, stop immediately
		for _, res := range runResp {
			status := mapJudgeStatus(res.Status, res.ExitStatus)
			if status != "ACCEPTED" {
				c.logger.Info("early termination: non-ACCEPTED result detected, skipping remaining batches",
					zap.Int("tests_run", len(allResponses)),
					zap.Int("tests_total", bundle.TestCount),
					zap.String("failing_status", status),
				)
				return c.parseJudgeResult(allResponses, bundle), nil
			}
		}
	}

	c.logger.Info("go-judge run completed", zap.Int("total_tests", len(allResponses)))

	return c.parseJudgeResult(allResponses, bundle), nil
}

// parseJudgeResult processes go-judge responses and compares output with expected.
// For failed tests, reads input and expected output from disk to include in the result.
func (c *GoJudgeClient) parseJudgeResult(responses gojudge.Response, bundle *outbound.TestCaseBundle) *outbound.ExecutionResult {
	result := &outbound.ExecutionResult{
		Status:        "ACCEPTED",
		ExecutionTime: 0,
		MemoryUsed:    0,
		TestCases:     make([]outbound.TestCaseResult, 0, len(responses)),
	}

	maxTime := 0
	maxMem := 0
	allAccepted := true
	firstFailIndex := -1

	for i, res := range responses {
		testIndex := i + 1

		status := mapJudgeStatus(res.Status, res.ExitStatus)
		if status != "ACCEPTED" && firstFailIndex == -1 {
			allAccepted = false
			firstFailIndex = i
		}

		if int(res.Time/1000000) > maxTime {
			maxTime = int(res.Time / 1000000)
		}
		if int(res.Memory/1024) > maxMem {
			maxMem = int(res.Memory / 1024)
		}

		var actualOutput *string
		if out, ok := res.Files["stdout"]; ok {
			actualOutput = &out
		}

		result.TestCases = append(result.TestCases, outbound.TestCaseResult{
			Index:         testIndex,
			Status:        status,
			ActualOutput:  actualOutput,
			ExecutionTime: int(res.Time / 1000000),
			MemoryUsed:    int(res.Memory / 1024),
		})
	}

	if !allAccepted {
		// Set overall status to first failure
		result.Status = result.TestCases[firstFailIndex].Status
		// Attach input + expected output for the first failed test
		c.attachTestData(&result.TestCases[firstFailIndex], bundle)
	} else {
		// All execution statuses are ACCEPTED — compare output with expected
		for i, tcRes := range result.TestCases {
			expectedPath := filepath.Join(bundle.Dir, fmt.Sprintf("%d.out", tcRes.Index))
			expectedBytes, err := os.ReadFile(expectedPath)
			if err != nil {
				c.logger.Error("failed to read expected output",
					zap.Int("test_index", tcRes.Index),
					zap.Error(err),
				)
				result.TestCases[i].Status = "SYSTEM_ERROR"
				allAccepted = false
				if firstFailIndex == -1 {
					firstFailIndex = i
				}
				continue
			}

			expected := strings.TrimRight(string(expectedBytes), "\n\r ")
			actual := ""
			if tcRes.ActualOutput != nil {
				actual = strings.TrimRight(*tcRes.ActualOutput, "\n\r ")
			}

			if actual != expected {
				result.TestCases[i].Status = "WRONG_ANSWER"
				expectedStr := expected
				result.TestCases[i].ExpectedOutput = &expectedStr
				allAccepted = false
				if firstFailIndex == -1 {
					firstFailIndex = i
				}
			}
		}
		if !allAccepted {
			result.Status = result.TestCases[firstFailIndex].Status
			c.attachTestData(&result.TestCases[firstFailIndex], bundle)
		}
	}

	result.ExecutionTime = maxTime
	result.MemoryUsed = maxMem

	return result
}

// attachTestData reads input and expected output files for a failed test case.
// Data is truncated to 10KB to avoid bloating Kafka messages.
func (c *GoJudgeClient) attachTestData(tc *outbound.TestCaseResult, bundle *outbound.TestCaseBundle) {
	const maxSize = 10 * 1024 // 10KB

	// Read input
	inputPath := filepath.Join(bundle.Dir, fmt.Sprintf("%d.in", tc.Index))
	if data, err := os.ReadFile(inputPath); err == nil {
		s := string(data)
		if len(s) > maxSize {
			s = s[:maxSize] + "...(truncated)"
		}
		tc.Input = &s
		c.logger.Info("attachTestData: input read OK", zap.Int("index", tc.Index), zap.Int("len", len(s)))
	} else {
		c.logger.Error("attachTestData: failed to read input", zap.String("path", inputPath), zap.Error(err))
	}

	// Read expected output (if not already set)
	if tc.ExpectedOutput == nil {
		expectedPath := filepath.Join(bundle.Dir, fmt.Sprintf("%d.out", tc.Index))
		if data, err := os.ReadFile(expectedPath); err == nil {
			s := strings.TrimRight(string(data), "\n\r ")
			if len(s) > maxSize {
				s = s[:maxSize] + "...(truncated)"
			}
			tc.ExpectedOutput = &s
			c.logger.Info("attachTestData: expected read OK", zap.Int("index", tc.Index), zap.Int("len", len(s)))
		} else {
			c.logger.Error("attachTestData: failed to read expected", zap.String("path", expectedPath), zap.Error(err))
		}
	} else {
		c.logger.Info("attachTestData: expected already set", zap.Int("index", tc.Index), zap.Int("len", len(*tc.ExpectedOutput)))
	}

	// Truncate actual output if needed
	if tc.ActualOutput != nil && len(*tc.ActualOutput) > maxSize {
		s := (*tc.ActualOutput)[:maxSize] + "...(truncated)"
		tc.ActualOutput = &s
	}

	c.logger.Info("attachTestData: final state",
		zap.Int("index", tc.Index),
		zap.Bool("has_input", tc.Input != nil),
		zap.Bool("has_expected", tc.ExpectedOutput != nil),
		zap.Bool("has_actual", tc.ActualOutput != nil),
	)
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
	case "File Error", "Nonzero Exit Status", "Signalled", "Run Error":
		return "RUNTIME_ERROR"
	case "Internal Error":
		return "SYSTEM_ERROR"
	default:
		return "SYSTEM_ERROR"
	}
}

func stringPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}
