package testcase

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
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
	if len(got) != 2 {
		t.Fatalf("len(testCases) = %d, want 2", len(got))
	}
	if got[0].Index != 1 || got[0].ID != "1" || got[0].Stdin != "1\n" || *got[0].ExpectedOutput != "1\n" {
		t.Fatalf("first testcase = %#v", got[0])
	}
	if got[1].Index != 2 || got[1].ID != "2" || got[1].Stdin != "2\n" || *got[1].ExpectedOutput != "2\n" {
		t.Fatalf("second testcase = %#v", got[1])
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
