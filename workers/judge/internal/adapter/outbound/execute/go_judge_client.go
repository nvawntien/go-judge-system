package execute

import (
	"context"
	"fmt"
	"time"

	"go-judge-system/pkg/gojudge"
	"go-judge-system/workers/judge/internal/application/port/outbound"

	resty "github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

// GoJudgeClient executes code using go-judge service
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

func (c *GoJudgeClient) Execute(ctx context.Context, language, sourceCode string, testCases []outbound.TestCase) (*outbound.ExecutionResult, error) {
	if language == "" {
		return nil, fmt.Errorf("language not specified")
	}

	if sourceCode == "" {
		return nil, fmt.Errorf("source code is empty")
	}

	if len(testCases) == 0 {
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
						{Content: stringPtr("")}, // stdin
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
			// Compilation failed
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

		// Extract compiled binary file ID
		fileID, ok := res.FileIDs[gojudge.GetExeFileName(language)]
		if !ok {
			return nil, fmt.Errorf("compile succeeded but exe fileId not found in response")
		}
		exeFileID = fileID
	}

	// Step 2: Execution for each testcase
	runReq := gojudge.Request{
		Cmd: make([]*gojudge.Cmd, 0, len(testCases)),
	}

	for _, tc := range testCases {
		runCmd := &gojudge.Cmd{
			Args: langCfg.Run.Command,
			Env:  langCfg.Run.Env,
			Files: []*gojudge.File{
				{Content: stringPtr(tc.Input)}, // stdin
				{Name: stringPtr("stdout"), Max: int64Ptr(10485760)}, // stdout
				{Name: stringPtr("stderr"), Max: int64Ptr(10485760)}, // stderr
			},
			CopyOut:     []string{"stdout"},
			MemoryLimit: runMemLimit,
			CPULimit:    runTimeLimit,
			ProcLimit:   50, // Let the Go Runtime spawn threads
		}

		if hasCompile {
			runCmd.CopyIn = map[string]*gojudge.File{
				gojudge.GetExeFileName(language): {FileID: stringPtr(exeFileID)},
			}
		} else {
			// Interpreted languages pass source directly
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

	if resp.IsError() || len(runResp) != len(testCases) {
		return nil, fmt.Errorf("go-judge run returned status: %d or mismatched response length", resp.StatusCode())
	}
	
	c.logger.Info("go-judge run raw status", zap.Any("resp", runResp))

	return parseJudgeResult(runResp, testCases), nil
}

func parseJudgeResult(responses gojudge.Response, testCases []outbound.TestCase) *outbound.ExecutionResult {
	result := &outbound.ExecutionResult{
		Status:        "ACCEPTED",
		ExecutionTime: 0,
		MemoryUsed:    0,
		TestCases:     make([]outbound.TestCaseResult, 0, len(testCases)),
	}

	maxTime := 0
	maxMem := 0
	allAccepted := true

	for i, res := range responses {
		tc := testCases[i]
		
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
			TestCaseID:    tc.ID,
			Status:        status,
			ActualOutput:  actualOutput,
			ExecutionTime: int(res.Time / 1000000), // ns to ms
			MemoryUsed:    int(res.Memory / 1024),  // bytes to KB
			Order:         tc.Order,
		})
	}

	if !allAccepted {
		// Just find the first failed test case status to represent the overall status
		for _, tc := range result.TestCases {
			if tc.Status != "ACCEPTED" {
				result.Status = tc.Status
				break
			}
		}
	} else {
		// Verify expected output
		for i, tcRes := range result.TestCases {
			expected := testCases[i].Output
			actual := ""
			if tcRes.ActualOutput != nil {
				actual = *tcRes.ActualOutput
			}
			
			if actual != expected { // NOTE: Should use right trim for strict matching
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
