package testcase

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-judge-system/workers/judge/internal/application/port/outbound"
	workerdomain "go-judge-system/workers/judge/internal/domain"

	"go.uber.org/zap"
)

const (
	defaultCacheBaseDir      = "/cache/testcases"
	defaultHTTPTimeout       = 30 * time.Second
	defaultMaxZipBytes       = 64 * 1024 * 1024
	defaultMaxExtractedBytes = 128 * 1024 * 1024
	cacheVersionFile         = ".version"
	cacheSHA256File          = ".sha256"
	cacheZipFile             = "bundle.zip"
)

var (
	testcaseFilePattern = regexp.MustCompile(`^0*([1-9][0-9]*)\.(in|out)$`)
	sha256HexPattern    = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type OfficialLoader struct {
	httpClient        *http.Client
	cacheBaseDir      string
	maxZipBytes       int64
	maxExtractedBytes int64
	logger            *zap.Logger
}

func NewOfficialLoader(logger *zap.Logger) *OfficialLoader {
	return &OfficialLoader{
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
		cacheBaseDir:      defaultCacheBaseDir,
		maxZipBytes:       defaultMaxZipBytes,
		maxExtractedBytes: defaultMaxExtractedBytes,
		logger:            logger,
	}
}

func (l *OfficialLoader) Load(
	ctx context.Context,
	metadata outbound.ProblemTestCaseMetadata,
) (outbound.OfficialTestCaseBundle, error) {
	if err := validateMetadata(metadata); err != nil {
		return outbound.OfficialTestCaseBundle{}, err
	}

	cacheDir := filepath.Join(l.cacheBaseDir, fmt.Sprintf("problem_%d", metadata.ProblemID))
	version := strconv.Itoa(metadata.Version)
	versionFile := filepath.Join(cacheDir, cacheVersionFile)
	if cachedVersion, err := os.ReadFile(versionFile); err == nil && strings.TrimSpace(string(cachedVersion)) == version {
		if checksum, ok := l.verifiedCachedChecksum(cacheDir); ok {
			l.logger.Debug(
				"official testcase cache hit",
				zap.Int64("problem_id", metadata.ProblemID),
				zap.Int("version", metadata.Version),
			)
			return l.bundleFromDir(cacheDir, metadata, checksum)
		}
		l.logger.Debug(
			"official testcase cache missing or mismatched checksum, refreshing bundle",
			zap.Int64("problem_id", metadata.ProblemID),
			zap.Int("version", metadata.Version),
		)
	}

	l.logger.Info(
		"official testcase cache miss, downloading bundle",
		zap.Int64("problem_id", metadata.ProblemID),
		zap.Int("version", metadata.Version),
	)

	zipPath := filepath.Join(os.TempDir(), fmt.Sprintf("tc_%d_%d_%s.zip", metadata.ProblemID, metadata.Version, randHex(6)))
	checksum, err := l.downloadToFile(ctx, metadata.ZipDownloadURL, zipPath)
	if err != nil {
		return outbound.OfficialTestCaseBundle{}, fmt.Errorf("download testcase bundle: %w", err)
	}
	defer func() {
		if err := os.Remove(zipPath); err != nil && !os.IsNotExist(err) {
			l.logger.Debug("failed to remove temp testcase zip", zap.String("path", zipPath), zap.Error(err))
		}
	}()

	tmpDir := fmt.Sprintf("%s_tmp_%s", cacheDir, randHex(8))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return outbound.OfficialTestCaseBundle{}, fmt.Errorf("create testcase cache temp dir: %w", err)
	}
	removeTmp := true
	defer func() {
		if removeTmp {
			if err := os.RemoveAll(tmpDir); err != nil {
				l.logger.Debug("failed to remove temp testcase dir", zap.String("path", tmpDir), zap.Error(err))
			}
		}
	}()

	if err := l.extractZip(zipPath, tmpDir); err != nil {
		return outbound.OfficialTestCaseBundle{}, workerdomain.MarkNonRetryable(fmt.Errorf("extract testcase bundle: %w", err))
	}
	if _, err := l.loadFromDir(tmpDir, metadata.TestCount); err != nil {
		return outbound.OfficialTestCaseBundle{}, err
	}
	if err := copyFileAtomic(zipPath, filepath.Join(tmpDir, cacheZipFile)); err != nil {
		return outbound.OfficialTestCaseBundle{}, fmt.Errorf("write testcase cache zip: %w", err)
	}
	if err := writeFileAtomic(filepath.Join(tmpDir, cacheVersionFile), []byte(version), 0644); err != nil {
		return outbound.OfficialTestCaseBundle{}, fmt.Errorf("write testcase cache version: %w", err)
	}
	if err := writeFileAtomic(filepath.Join(tmpDir, cacheSHA256File), []byte(checksum), 0644); err != nil {
		return outbound.OfficialTestCaseBundle{}, fmt.Errorf("write testcase cache checksum: %w", err)
	}

	if err := os.MkdirAll(l.cacheBaseDir, 0755); err != nil {
		return outbound.OfficialTestCaseBundle{}, fmt.Errorf("create testcase cache base dir: %w", err)
	}
	if err := os.RemoveAll(cacheDir); err != nil {
		return outbound.OfficialTestCaseBundle{}, fmt.Errorf("remove stale testcase cache: %w", err)
	}
	if err := os.Rename(tmpDir, cacheDir); err != nil {
		return outbound.OfficialTestCaseBundle{}, fmt.Errorf("promote testcase cache: %w", err)
	}
	removeTmp = false

	l.logger.Info(
		"official testcase bundle cached",
		zap.Int64("problem_id", metadata.ProblemID),
		zap.Int("version", metadata.Version),
		zap.Int("test_count", metadata.TestCount),
	)
	return l.bundleFromDir(cacheDir, metadata, checksum)
}

func (l *OfficialLoader) verifiedCachedChecksum(cacheDir string) (string, bool) {
	checksumBytes, err := os.ReadFile(filepath.Join(cacheDir, cacheSHA256File))
	if err != nil {
		return "", false
	}
	checksum := strings.TrimSpace(string(checksumBytes))
	if !sha256HexPattern.MatchString(checksum) {
		return "", false
	}

	actual, err := sha256File(filepath.Join(cacheDir, cacheZipFile))
	if err != nil {
		return "", false
	}
	if actual != checksum {
		return "", false
	}
	return checksum, true
}

func validateMetadata(metadata outbound.ProblemTestCaseMetadata) error {
	if metadata.ProblemID <= 0 {
		return workerdomain.MarkNonRetryable(fmt.Errorf("problem_id must be greater than zero"))
	}
	if strings.TrimSpace(metadata.ZipDownloadURL) == "" {
		return workerdomain.MarkNonRetryable(fmt.Errorf("zip_download_url is required"))
	}
	if metadata.TestCount <= 0 {
		return workerdomain.MarkNonRetryable(fmt.Errorf("test_count must be greater than zero"))
	}
	if metadata.Version <= 0 {
		return workerdomain.MarkNonRetryable(fmt.Errorf("version must be greater than zero"))
	}
	return nil
}

func (l *OfficialLoader) downloadToFile(ctx context.Context, url, destPath string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", workerdomain.MarkNonRetryable(fmt.Errorf("build testcase download request: %w", err))
	}

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("testcase download returned status %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	reader := io.LimitReader(resp.Body, l.maxZipBytes+1)
	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, hash), reader)
	if err != nil {
		return "", err
	}
	if n > l.maxZipBytes {
		return "", workerdomain.MarkNonRetryable(fmt.Errorf("testcase zip exceeds %d bytes", l.maxZipBytes))
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func sha256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmpPath := fmt.Sprintf("%s.tmp.%s", path, randHex(6))
	if err := os.WriteFile(tmpPath, data, perm); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func copyFileAtomic(srcPath, destPath string) error {
	tmpPath := fmt.Sprintf("%s.tmp.%s", destPath, randHex(6))
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dest, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dest, src); err != nil {
		_ = dest.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := dest.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func (l *OfficialLoader) bundleFromDir(
	dir string,
	metadata outbound.ProblemTestCaseMetadata,
	checksum string,
) (outbound.OfficialTestCaseBundle, error) {
	testCases, err := l.loadFromDir(dir, metadata.TestCount)
	if err != nil {
		return outbound.OfficialTestCaseBundle{}, err
	}
	return outbound.OfficialTestCaseBundle{
		TestCases:       testCases,
		TestCount:       metadata.TestCount,
		Version:         metadata.Version,
		DatasetChecksum: checksum,
	}, nil
}

func (l *OfficialLoader) extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	var extracted int64
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if f.FileInfo().Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("zip entry %q is a symlink", f.Name)
		}

		name := filepath.Clean(f.Name)
		if filepath.IsAbs(name) || strings.HasPrefix(name, ".."+string(os.PathSeparator)) || name == ".." {
			return fmt.Errorf("zip entry %q escapes destination", f.Name)
		}
		base := filepath.Base(name)
		if !testcaseFilePattern.MatchString(base) {
			continue
		}

		extracted += int64(f.UncompressedSize64)
		if extracted > l.maxExtractedBytes {
			return fmt.Errorf("testcase bundle exceeds %d extracted bytes", l.maxExtractedBytes)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open zip entry %s: %w", f.Name, err)
		}
		destPath := filepath.Join(destDir, base)
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

func (l *OfficialLoader) loadFromDir(dir string, testCount int) ([]outbound.ExecutionTestCase, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read testcase dir: %w", err)
	}

	inputs := make(map[int]string)
	outputs := make(map[int]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := testcaseFilePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}
		index, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, workerdomain.MarkNonRetryable(fmt.Errorf("parse testcase index %q: %w", entry.Name(), err))
		}
		path := filepath.Join(dir, entry.Name())
		switch matches[2] {
		case "in":
			inputs[index] = path
		case "out":
			outputs[index] = path
		}
	}

	indexes := make([]int, 0, len(inputs))
	for index := range inputs {
		if _, ok := outputs[index]; !ok {
			return nil, workerdomain.MarkNonRetryable(fmt.Errorf("missing output file for testcase %d", index))
		}
		indexes = append(indexes, index)
	}
	for index := range outputs {
		if _, ok := inputs[index]; !ok {
			return nil, workerdomain.MarkNonRetryable(fmt.Errorf("missing input file for testcase %d", index))
		}
	}
	sort.Ints(indexes)
	if len(indexes) != testCount {
		return nil, workerdomain.MarkNonRetryable(
			fmt.Errorf("testcase count mismatch: metadata=%d files=%d", testCount, len(indexes)),
		)
	}

	testCases := make([]outbound.ExecutionTestCase, 0, len(indexes))
	for sequence, index := range indexes {
		stdinBytes, err := os.ReadFile(inputs[index])
		if err != nil {
			return nil, fmt.Errorf("read input file for testcase %d: %w", index, err)
		}
		expectedBytes, err := os.ReadFile(outputs[index])
		if err != nil {
			return nil, fmt.Errorf("read output file for testcase %d: %w", index, err)
		}
		expected := string(expectedBytes)
		testCases = append(testCases, outbound.ExecutionTestCase{
			Index:          sequence + 1,
			ID:             strconv.Itoa(index),
			Kind:           "official",
			Stdin:          string(stdinBytes),
			ExpectedOutput: &expected,
		})
	}
	return testCases, nil
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}
