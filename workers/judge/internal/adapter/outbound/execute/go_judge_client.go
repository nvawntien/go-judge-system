package execute

import (
	"context"
	"fmt"

	"go-judge-system/workers/judge/internal/application/port/outbound"
	"go.uber.org/zap"
)

// GoJudgeClient executes code using go-judge service
type GoJudgeClient struct {
	baseURL string
	logger  *zap.Logger
}

func NewGoJudgeClient(baseURL string, logger *zap.Logger) *GoJudgeClient {
	return &GoJudgeClient{
		baseURL: baseURL,
		logger:  logger,
	}
}

// Execute submits code to go-judge service for compilation and execution
// This is a placeholder implementation. Real implementation would:
// 1. Call go-judge HTTP API with language, source code, and test cases
// 2. Parse results and map to ExecutionResult
// 3. Handle timeouts and errors appropriately
func (c *GoJudgeClient) Execute(ctx context.Context, language, sourceCode string, testCases []outbound.TestCase) (*outbound.ExecutionResult, error) {
	if language == "" {
		return nil, fmt.Errorf("language not specified")
	}

	if sourceCode == "" {
		return nil, fmt.Errorf("source code is empty")
	}

	c.logger.Info(
		"submitting code to go-judge",
		zap.String("language", language),
		zap.Int("test_cases", len(testCases)),
	)

	// TODO: Implement actual go-judge HTTP client
	// For MVP, return a placeholder result
	result := &outbound.ExecutionResult{
		Status:        "ACCEPTED",
		ExecutionTime: 100,
		MemoryUsed:    10240,
		TestCases:     []outbound.TestCaseResult{},
	}

	return result, nil
}
