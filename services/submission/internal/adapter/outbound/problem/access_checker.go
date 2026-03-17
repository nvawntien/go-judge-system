package problem

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/submission/internal/application/port/outbound"
)

type httpProblemAccessChecker struct {
	baseURL string
	client  *http.Client
}

func NewProblemAccessChecker() outbound.ProblemAccessChecker {
	baseURL := os.Getenv("PROBLEM_SERVICE_URL")
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "http://problem-service:8082"
	}

	return &httpProblemAccessChecker{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

func (c *httpProblemAccessChecker) CanManageProblem(ctx context.Context, claims auth.Claims, problemID int64) (bool, error) {
	if claims.IsSuperAdmin() {
		return true, nil
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/admin/problems/%d", c.baseURL, problemID),
		nil,
	)
	if err != nil {
		return false, err
	}

	req.Header.Set("X-User-ID", claims.UserID)
	req.Header.Set("X-Role", claims.Role)
	req.Header.Set("X-Username", claims.Username)

	resp, err := c.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusForbidden, http.StatusNotFound, http.StatusUnauthorized:
		return false, nil
	default:
		return false, fmt.Errorf("problem service unexpected status: %s", strconv.Itoa(resp.StatusCode))
	}
}
