package runner

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/config"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/model"
)

func writeCredentialFile(t *testing.T, token string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "users.json")
	content := fmt.Sprintf(`{"schema_version":1,"users":[{"alias":"bench-001","cookies":{"access_token":"%s"}}]}`, token)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func futureJWT() string {
	payload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"exp":%d}`, time.Now().Add(time.Hour).Unix())))
	return "header." + payload + ".signature"
}

func baseConfig(t *testing.T, rawURL, users, source string) config.Config {
	t.Helper()
	cfg := config.Defaults(config.ModeBurst)
	u, _ := url.Parse(rawURL)
	cfg.BaseURL = u
	cfg.BaseURLRaw = rawURL
	cfg.UsersFile = users
	cfg.ProblemID = 7
	cfg.ProblemSlug = "sample"
	cfg.Language = "GO"
	cfg.SourceFile = source
	cfg.ExpectedVerdict = "ACCEPTED"
	cfg.SubmitCooldown = time.Millisecond
	cfg.MaxSubmissions = 1
	cfg.BurstSize = 1
	cfg.SubmissionTimeout = time.Second
	cfg.DrainTimeout = time.Second
	cfg.WarmupTimeout = time.Second
	cfg.SSEConnectTimeout = time.Second
	cfg.SSEIdleTimeout = 100 * time.Millisecond
	cfg.SSEBackoffBase = time.Millisecond
	cfg.SSEBackoffMax = 2 * time.Millisecond
	cfg.SafetyReconcileInterval = 0
	cfg.ResultRoot = filepath.Join(t.TempDir(), "results")
	return cfg
}

func TestPreflightUsesGETOnly(t *testing.T) {
	var post atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			post.Add(1)
		}
		switch r.URL.Path {
		case "/api/v1/me":
			_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":{"id":"real-user-id","role":"user","is_active":true}}`))
		case "/api/v1/problems/sample":
			_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":{"id":7}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	source := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(source, []byte("source-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := baseConfig(t, server.URL, writeCredentialFile(t, "access-secret"), source)
	prepared, err := Preflight(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.SourceSHA256 == "" || post.Load() != 0 {
		t.Fatalf("prepared=%+v post=%d", prepared, post.Load())
	}
}

func TestRunMakesOnePOSTAndRedactsSource(t *testing.T) {
	var posts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/me":
			_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":{"id":"real-user-id","role":"user","is_active":true}}`))
		case "/api/v1/problems/sample":
			_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":{"id":7}}`))
		case "/api/v1/submissions":
			posts.Add(1)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":"success","code":20100,"data":{"id":9,"status":"PENDING"}}`))
		case "/api/v1/submissions/9/events/ticket":
			_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":{"ticket":"ticket-secret"}}`))
		case "/events/submissions/9":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprint(w, "event: submission.completed\ndata: {\"submission_id\":9,\"attempt_id\":\"attempt\",\"status\":\"ACCEPTED\"}\n\n")
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	source := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(source, []byte("source-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	accessToken := futureJWT()
	cfg := baseConfig(t, server.URL, writeCredentialFile(t, accessToken), source)
	prepared, err := Preflight(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), prepared)
	if err != nil {
		t.Fatal(err)
	}
	if posts.Load() != 1 || run.Summary.Counts.Accepted != 1 || run.Summary.Counts.Terminal != 1 {
		t.Fatalf("posts=%d summary=%+v", posts.Load(), run.Summary)
	}
	for _, name := range []string{"run.json", "summary.json", "submissions.csv", "windows.csv", "report.md"} {
		data, err := os.ReadFile(filepath.Join(run.Dir, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{"source-secret", "ticket-secret", "real-user-id", accessToken} {
			if string(data) != "" && containsText(string(data), secret) {
				t.Fatalf("%s leaked to %s", secret, name)
			}
		}
	}
}

func TestWarmupIsExcludedFromMeasuredCounts(t *testing.T) {
	var posts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/me":
			_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":{"id":"real-user-id","role":"user","is_active":true}}`))
		case "/api/v1/problems/sample":
			_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":{"id":7}}`))
		case "/api/v1/submissions":
			id := posts.Add(1)
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprintf(w, `{"status":"success","code":20100,"data":{"id":%d,"status":"PENDING"}}`, id)
		case "/api/v1/submissions/1/events/ticket", "/api/v1/submissions/2/events/ticket":
			_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":{"ticket":"ticket-secret"}}`))
		case "/events/submissions/1", "/events/submissions/2":
			w.Header().Set("Content-Type", "text/event-stream")
			id := 1
			if r.URL.Path == "/events/submissions/2" {
				id = 2
			}
			_, _ = fmt.Fprintf(w, "event: submission.completed\ndata: {\"submission_id\":%d,\"attempt_id\":\"attempt\",\"status\":\"ACCEPTED\"}\n\n", id)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	source := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(source, []byte("source-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := baseConfig(t, server.URL, writeCredentialFile(t, futureJWT()), source)
	cfg.WarmupCount = 1
	cfg.MaxSubmissions = 2
	prepared, err := Preflight(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), prepared)
	if err != nil {
		t.Fatal(err)
	}
	if posts.Load() != 2 || run.Summary.Counts.Accepted != 1 || run.Summary.Counts.Intended != 1 {
		t.Fatalf("posts=%d summary=%+v", posts.Load(), run.Summary)
	}
}

func TestBoundaryStraddlingHTTP201IsDrainWorkNotLoadAcceptance(t *testing.T) {
	start := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Second)
	id := int64(10)
	accepted := end // half-open load window excludes this response.
	terminal := end.Add(time.Second)
	cfg := config.Defaults(config.ModeSustained)
	cfg.RunID = "boundary"
	records := []model.SubmissionRecord{{Phase: model.PhaseLoad, IntendedAt: &start, PostStartedAt: &start, PostCompletedAt: &accepted, Accepted: true, SubmissionID: &id, TerminalObservedAt: &terminal, TerminalStatus: "ACCEPTED"}}
	summary := summarize(cfg, records, nil, start, end, end, terminal.Add(time.Second), map[string]struct{}{}, 1)
	if summary.Counts.Accepted != 1 || summary.LoadWindow.Accepted != 0 || summary.LoadWindow.BoundaryAcceptedAfterLoad != 1 || summary.LoadWindow.OutstandingAtEnd != 0 {
		t.Fatalf("load summary=%+v counts=%+v", summary.LoadWindow, summary.Counts)
	}
	if summary.Drain.Completed != 1 || summary.Drain.Remaining != 0 {
		t.Fatalf("drain=%+v", summary.Drain)
	}
}

func containsText(value, needle string) bool {
	return len(needle) > 0 && len(value) >= len(needle) && func() bool {
		for i := 0; i+len(needle) <= len(value); i++ {
			if value[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	}()
}
