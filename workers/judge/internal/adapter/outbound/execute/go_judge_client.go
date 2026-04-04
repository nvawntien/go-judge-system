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
		client: resty.New().SetBaseURL(baseURL).SetTimeout(30 * time.Second),
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
						{Content: stringPtr("")},                          // stdin
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

	// Step 2: Build run commands for all test cases
	// Key change: stdin uses Src field (file path on shared volume) instead of Content (string in RAM)
	runReq := gojudge.Request{
		Cmd: make([]*gojudge.Cmd, 0, bundle.TestCount),
	}

	for i := 1; i <= bundle.TestCount; i++ {
		// Path to .in file — go-judge reads DIRECTLY from shared volume
		// Worker does NOT need to load .in file into RAM!
		inputPath := filepath.Join(bundle.Dir, fmt.Sprintf("%d.in", i))

		runCmd := &gojudge.Cmd{
			Args: langCfg.Run.Command,
			Env:  langCfg.Run.Env,
			Files: []*gojudge.File{
				{Src: stringPtr(inputPath)},                             // stdin: go-judge reads file directly
				{Name: stringPtr("stdout"), Max: int64Ptr(10485760)},     // stdout
				{Name: stringPtr("stderr"), Max: int64Ptr(10485760)},     // stderr
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
		c.logger.Error("failed to call go-judge API for execution", zap.Error(err))
		return nil, fmt.Errorf("call go-judge run API: %w", err)
	}

	if resp.IsError() || len(runResp) != bundle.TestCount {
		return nil, fmt.Errorf("go-judge run returned status: %d or mismatched response length (expected %d, got %d)",
			resp.StatusCode(), bundle.TestCount, len(runResp))
	}

	c.logger.Info("go-judge run raw status", zap.Any("resp", runResp))

	return c.parseJudgeResult(runResp, bundle), nil
}

// parseJudgeResult processes go-judge responses and compares output with expected.
// Reads expected output from disk (only the .out files need to be read into RAM for comparison).
func (c *GoJudgeClient) parseJudgeResult(responses gojudge.Response, bundle *outbound.TestCaseBundle) *outbound.ExecutionResult {
	result := &outbound.ExecutionResult{
		Status:        "ACCEPTED",
		ExecutionTime: 0,
		MemoryUsed:    0,
		TestCases:     make([]outbound.TestCaseResult, 0, bundle.TestCount),
	}

	maxTime := 0
	maxMem := 0
	allAccepted := true

	for i, res := range responses {
		testIndex := i + 1

		status := mapJudgeStatus(res.Status, res.ExitStatus)
		if status != "ACCEPTED" {
			allAccepted = false
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
		for _, tc := range result.TestCases {
			if tc.Status != "ACCEPTED" {
				result.Status = tc.Status
				break
			}
		}
	} else {
		// Compare actual output with expected output from disk
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
				continue
			}

			expected := strings.TrimRight(string(expectedBytes), "\n\r ")
			actual := ""
			if tcRes.ActualOutput != nil {
				actual = strings.TrimRight(*tcRes.ActualOutput, "\n\r ")
			}

			if actual != expected {
				result.TestCases[i].Status = "WRONG_ANSWER"
				allAccepted = false
			}
		}
		if !allAccepted {
			result.Status = "WRONG_ANSWER"
		}
	}

	result.ExecutionTime = maxTime
	result.MemoryUsed = maxMem

	return result
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
	case "File Error", "Non Zero Exit Status", "Signalled", "Run Error":
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
