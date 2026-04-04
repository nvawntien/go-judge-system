package problem

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-judge-system/workers/judge/internal/application/port/outbound"

	"go.uber.org/zap"
)

const cacheBaseDir = "/cache/testcases"

// ProblemServiceClient fetches test cases from Problem Service's internal API.
// Implements local disk cache with atomic rename to prevent cache stampede.
type ProblemServiceClient struct {
	httpClient *http.Client
	baseURL    string
	logger     *zap.Logger
}

func NewProblemServiceClient(baseURL string, logger *zap.Logger) *ProblemServiceClient {
	return &ProblemServiceClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
		logger:  logger,
	}
}

// internalTestCaseResponse matches the Problem Service internal API response.
type internalTestCaseResponse struct {
	Code int `json:"code"`
	Data struct {
		ProblemID      int64  `json:"problem_id"`
		TestCount      int    `json:"test_count"`
		Version        string `json:"version"`
		ZipDownloadURL string `json:"zip_download_url"`
	} `json:"data"`
}

func (c *ProblemServiceClient) FetchTestCases(ctx context.Context, problemID int64) (*outbound.TestCaseBundle, error) {
	// 1. Call Problem Service internal API → get metadata + presigned URL
	meta, err := c.fetchMetadata(ctx, problemID)
	if err != nil {
		return nil, fmt.Errorf("fetch testcase metadata: %w", err)
	}

	// 2. Check local cache
	cacheDir := filepath.Join(cacheBaseDir, fmt.Sprintf("problem_%d", problemID))
	versionFile := filepath.Join(cacheDir, ".version")

	cachedVersion, readErr := os.ReadFile(versionFile)
	if readErr == nil && strings.TrimSpace(string(cachedVersion)) == meta.Data.Version {
		// CACHE HIT — use files on disk, zero network I/O
		c.logger.Debug("testcase cache hit",
			zap.Int64("problem_id", problemID),
			zap.String("version", meta.Data.Version),
		)
		return &outbound.TestCaseBundle{Dir: cacheDir, TestCount: meta.Data.TestCount}, nil
	}

	// 3. CACHE MISS — download ZIP from MinIO + atomic rename
	c.logger.Info("testcase cache miss, downloading",
		zap.Int64("problem_id", problemID),
		zap.String("version", meta.Data.Version),
	)

	// 3a. Download ZIP stream → temp file (not in RAM)
	zipPath := filepath.Join(os.TempDir(), fmt.Sprintf("tc_%d_%s_%s.zip", problemID, meta.Data.Version, randHex(6)))
	if err := c.downloadToFile(ctx, meta.Data.ZipDownloadURL, zipPath); err != nil {
		return nil, fmt.Errorf("download zip: %w", err)
	}
	defer os.Remove(zipPath)

	// 3b. Extract into TEMP directory with random suffix (NOT the target!)
	tmpDir := fmt.Sprintf("%s_tmp_%s", cacheDir, randHex(8))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, fmt.Errorf("create tmp dir: %w", err)
	}
	if err := extractZip(zipPath, tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("extract zip: %w", err)
	}

	// 3c. Write version file INTO temp directory
	if err := os.WriteFile(filepath.Join(tmpDir, ".version"), []byte(meta.Data.Version), 0644); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("write version file: %w", err)
	}

	// 3d. ATOMIC RENAME — place temp directory at target position
	//     Remove old cache (if any), then rename
	os.RemoveAll(cacheDir) // ignore error: directory may not exist
	if err := os.Rename(tmpDir, cacheDir); err != nil {
		// Race condition: another goroutine already renamed successfully
		// → Cleanup our temp dir, use winner's cache
		c.logger.Debug("cache rename race lost, using winner's cache",
			zap.Int64("problem_id", problemID),
		)
		os.RemoveAll(tmpDir)

		// Verify winner's cache has correct version
		winnerVersion, readErr := os.ReadFile(filepath.Join(cacheDir, ".version"))
		if readErr != nil || strings.TrimSpace(string(winnerVersion)) != meta.Data.Version {
			return nil, fmt.Errorf("cache race: winner has wrong version")
		}
	}

	c.logger.Info("testcase cached successfully",
		zap.Int64("problem_id", problemID),
		zap.String("version", meta.Data.Version),
		zap.Int("test_count", meta.Data.TestCount),
	)

	return &outbound.TestCaseBundle{Dir: cacheDir, TestCount: meta.Data.TestCount}, nil
}

// fetchMetadata calls the Problem Service internal API.
func (c *ProblemServiceClient) fetchMetadata(ctx context.Context, problemID int64) (*internalTestCaseResponse, error) {
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

	var apiResp internalTestCaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &apiResp, nil
}

// downloadToFile streams a URL to a local file without loading into RAM.
func (c *ProblemServiceClient) downloadToFile(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// extractZip extracts all files from a ZIP archive to destDir.
func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		// Security: prevent path traversal
		name := filepath.Base(f.Name)
		destPath := filepath.Join(destDir, name)

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open zip entry %s: %w", f.Name, err)
		}

		outFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return fmt.Errorf("create file %s: %w", destPath, err)
		}

		if _, err := io.Copy(outFile, rc); err != nil {
			outFile.Close()
			rc.Close()
			return fmt.Errorf("extract %s: %w", f.Name, err)
		}

		outFile.Close()
		rc.Close()
	}

	return nil
}

// randHex generates a random hex string for unique temp directory names.
func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
