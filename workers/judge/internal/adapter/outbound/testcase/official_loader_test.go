package testcase

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"go-judge-system/workers/judge/internal/application/port/outbound"
	workerdomain "go-judge-system/workers/judge/internal/domain"

	"go.uber.org/zap"
)

func TestOfficialLoaderLoadsZeroPaddedPairsDeterministically(t *testing.T) {
	t.Parallel()

	zipBytes := buildZip(t, map[string]string{
		"002.in":  "2\n",
		"002.out": "2\n",
		"001.in":  "1\n",
		"001.out": "1\n",
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(zipBytes)
	}))
	t.Cleanup(server.Close)

	loader := NewOfficialLoader(zap.NewNop())
	loader.cacheBaseDir = t.TempDir()

	got, err := loader.Load(context.Background(), outbound.ProblemTestCaseMetadata{
		ProblemID:      42,
		ZipDownloadURL: server.URL,
		TestCount:      2,
		Version:        1,
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.TestCount != 2 {
		t.Fatalf("TestCount = %d, want 2", got.TestCount)
	}
	if got.Version != 1 {
		t.Fatalf("Version = %d, want 1", got.Version)
	}
	if got.DatasetChecksum == "" {
		t.Fatal("DatasetChecksum is empty")
	}
	if len(got.TestCases) != 2 {
		t.Fatalf("len(testCases) = %d, want 2", len(got.TestCases))
	}
	if got.TestCases[0].Index != 1 || got.TestCases[0].ID != "1" || got.TestCases[0].Stdin != "1\n" || *got.TestCases[0].ExpectedOutput != "1\n" {
		t.Fatalf("first testcase = %#v", got.TestCases[0])
	}
	if got.TestCases[1].Index != 2 || got.TestCases[1].ID != "2" || got.TestCases[1].Stdin != "2\n" || *got.TestCases[1].ExpectedOutput != "2\n" {
		t.Fatalf("second testcase = %#v", got.TestCases[1])
	}
}

func TestOfficialLoaderCacheHitMissProvenance(t *testing.T) {
	t.Parallel()

	zipBytes := buildZip(t, map[string]string{
		"001.in":  "1\n",
		"001.out": "1\n",
	})
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = w.Write(zipBytes)
	}))
	t.Cleanup(server.Close)

	loader := NewOfficialLoader(zap.NewNop())
	loader.cacheBaseDir = t.TempDir()
	metadata := outbound.ProblemTestCaseMetadata{
		ProblemID:      42,
		ZipDownloadURL: server.URL,
		TestCount:      1,
		Version:        1,
	}

	miss, err := loader.Load(context.Background(), metadata)
	if err != nil {
		t.Fatalf("cache miss Load() error = %v", err)
	}
	hit, err := loader.Load(context.Background(), metadata)
	if err != nil {
		t.Fatalf("cache hit Load() error = %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("HTTP requests = %d, want 1 cache miss only", requests.Load())
	}
	if miss.DatasetChecksum == "" || miss.DatasetChecksum != hit.DatasetChecksum {
		t.Fatalf("cache miss/hit checksums = %q/%q, want matching non-empty", miss.DatasetChecksum, hit.DatasetChecksum)
	}

	changedZipBytes := buildZip(t, map[string]string{
		"001.in":  "2\n",
		"001.out": "2\n",
	})
	changedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(changedZipBytes)
	}))
	t.Cleanup(changedServer.Close)

	changed, err := loader.Load(context.Background(), outbound.ProblemTestCaseMetadata{
		ProblemID:      42,
		ZipDownloadURL: changedServer.URL,
		TestCount:      1,
		Version:        2,
	})
	if err != nil {
		t.Fatalf("changed artifact Load() error = %v", err)
	}
	if changed.DatasetChecksum == miss.DatasetChecksum {
		t.Fatalf("changed artifact checksum = %q, want different from %q", changed.DatasetChecksum, miss.DatasetChecksum)
	}
}

func TestOfficialLoaderRejectsMismatchedPairsAsNonRetryable(t *testing.T) {
	t.Parallel()

	zipBytes := buildZip(t, map[string]string{
		"001.in": "1\n",
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(zipBytes)
	}))
	t.Cleanup(server.Close)

	loader := NewOfficialLoader(zap.NewNop())
	loader.cacheBaseDir = t.TempDir()

	_, err := loader.Load(context.Background(), outbound.ProblemTestCaseMetadata{
		ProblemID:      42,
		ZipDownloadURL: server.URL,
		TestCount:      1,
		Version:        1,
	})
	if err == nil {
		t.Fatal("Load() error = nil")
	}
	if !workerdomain.IsNonRetryable(err) {
		t.Fatalf("IsNonRetryable(%v) = false, want true", err)
	}
}

func buildZip(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}
