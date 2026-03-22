package problem

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go-judge-system/workers/judge/internal/application/port/outbound"

	"go.uber.org/zap"
)

// ProblemServiceClient fetches test cases from Problem Service's internal API.
// This client communicates over the Docker internal network (judge_internal).
type ProblemServiceClient struct {
	httpClient *http.Client
	baseURL    string
	logger     *zap.Logger
}

func NewProblemServiceClient(baseURL string, logger *zap.Logger) *ProblemServiceClient {
	return &ProblemServiceClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: baseURL,
		logger:  logger,
	}
}

type testCaseAPIResponse struct {
	ProblemID int64                  `json:"problem_id"`
	TestCases []testCaseAPIItem      `json:"test_cases"`
	Total     int                    `json:"total"`
}

type testCaseAPIItem struct {
	ID             int64  `json:"id"`
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
	Order          int    `json:"order"`
}

func (c *ProblemServiceClient) FetchTestCases(ctx context.Context, problemID int64) ([]outbound.TestCase, error) {
	url := fmt.Sprintf("%s/internal/v1/problems/%d/testcases", c.baseURL, problemID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call problem service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("problem service returned status %d for problem_id=%d", resp.StatusCode, problemID)
	}

	var apiResp testCaseAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode problem service response: %w", err)
	}

	testCases := make([]outbound.TestCase, 0, len(apiResp.TestCases))
	for _, item := range apiResp.TestCases {
		testCases = append(testCases, outbound.TestCase{
			ID:     item.ID,
			Input:  item.Input,
			Output: item.ExpectedOutput,
			Order:  item.Order,
		})
	}

	c.logger.Debug("fetched test cases from problem service",
		zap.Int64("problem_id", problemID),
		zap.Int("count", len(testCases)),
	)

	return testCases, nil
}
