package runner

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/config"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/credentials"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/model"
)

func writeCredentialFile(t *testing.T, token string) string {
	t.Helper()
	return writeCredentialPool(t, []credentials.User{{Alias: "bench-001", Cookies: credentials.Cookies{AccessToken: token}}})
}

func writeCredentialPool(t *testing.T, users []credentials.User) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "users.json")
	content, err := json.Marshal(credentials.File{SchemaVersion: credentials.SchemaVersion, Users: users})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func futureJWT() string {
	return jwtExp(time.Now().Add(time.Hour))
}

func jwtExp(expiresAt time.Time) string {
	payload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"exp":%d}`, expiresAt.Unix())))
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
	cfg := baseConfig(t, server.URL, writeCredentialFile(t, futureJWT()), source)
	prepared, err := Preflight(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.SourceSHA256 == "" || post.Load() != 0 {
		t.Fatalf("prepared=%+v post=%d", prepared, post.Load())
	}
}

func TestRunMetadataCopiesOnlyValidatedSystemConfig(t *testing.T) {
	cfg := config.Defaults(config.ModeBurst)
	cfg.BaseURL, _ = url.Parse("https://benchmark.invalid")
	cfg.SystemConfig = &model.SystemConfig{
		Label: "pool-1", Release: "v-test",
		App:   model.NodeConfig{Nodes: 1, CPUCoresPerNode: 4, MemoryMiBPerNode: 4096},
		Judge: model.JudgeNodeConfig{Nodes: 1, CPUCoresPerNode: 2, MemoryMiBPerNode: 2048, WorkerPoolSize: 1, WorkerMemoryLimitMiB: 512, SandboxMemoryLimitMiB: 1024},
	}
	metadata := runMetadata(cfg, "source-hash", 1)
	if metadata.SystemConfig == nil || metadata.SystemConfig.Label != "pool-1" || metadata.SystemConfig.Judge.WorkerPoolSize != 1 {
		t.Fatalf("system config was not retained in run metadata: %+v", metadata.SystemConfig)
	}
}

func TestMassiveBurstMetadataRecordsCountsWithoutUserIdentities(t *testing.T) {
	cfg := config.Defaults(config.ModeBurst)
	cfg.BaseURL, _ = url.Parse("https://benchmark.invalid")
	cfg.Objective = config.ObjectiveMassiveBurst
	cfg.BurstSize, cfg.UserCount, cfg.MaxInFlight, cfg.MaxSubmissions = 1000, 1000, 1000, 1000
	metadata := runMetadata(cfg, "source-hash", 1)
	metadata.Users = model.UserSet{Configured: 10000, Selected: 1000, OneSubmitPerUser: true}
	metadata.BenchmarkObjective = string(cfg.Objective)
	if metadata.BenchmarkObjective != "massive-burst" || metadata.Users.Configured != 10000 || metadata.Users.Selected != 1000 || !metadata.Users.OneSubmitPerUser {
		t.Fatalf("massive burst metadata=%+v", metadata)
	}
	encoded, err := json.Marshal(metadata)
	if err != nil || strings.Contains(string(encoded), "bench-001") || strings.Contains(string(encoded), "benchmark_judge_") {
		t.Fatalf("metadata leaked benchmark identity: %s err=%v", encoded, err)
	}
}

func TestMassiveBurstSessionHorizonIncludesBoundedPoolPreflight(t *testing.T) {
	cfg := config.Defaults(config.ModeBurst)
	cfg.WarmupCount = 0
	cfg.APITimeout = 5 * time.Second
	cfg.PreflightConcurrency = 256
	cfg.BurstStartTimeout = 2 * time.Minute
	cfg.AuthValidityMargin = time.Minute
	// ceil(10000 / 256) == 40; each slot reserves refresh plus /me.
	if got, want := requiredSessionHorizonFor(cfg, 10000), 580*time.Second; got != want {
		t.Fatalf("horizon=%s want=%s", got, want)
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

func TestExactVolumeRunWritesMeasuredIntentAndExcludesWarmup(t *testing.T) {
	var posts atomic.Int64
	firstToken := futureJWT()
	secondToken := jwtExp(time.Now().Add(2 * time.Hour))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/api/v1/me":
			cookie, err := request.Cookie("access_token")
			if err != nil {
				t.Fatal(err)
			}
			id := "real-user-1"
			if cookie.Value == secondToken {
				id = "real-user-2"
			}
			_, _ = fmt.Fprintf(w, `{"status":"success","code":20000,"data":{"id":"%s","role":"user","is_active":true}}`, id)
		case request.URL.Path == "/api/v1/problems/sample":
			_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":{"id":7}}`))
		case request.Method == http.MethodPost && request.URL.Path == "/api/v1/submissions":
			id := posts.Add(1)
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprintf(w, `{"status":"success","code":20100,"data":{"id":%d,"status":"PENDING"}}`, id)
		case request.Method == http.MethodPost && strings.HasPrefix(request.URL.Path, "/api/v1/submissions/") && strings.HasSuffix(request.URL.Path, "/events/ticket"):
			_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":{"ticket":"test-ticket"}}`))
		case strings.HasPrefix(request.URL.Path, "/events/submissions/"):
			var id int64
			if _, err := fmt.Sscanf(strings.TrimPrefix(request.URL.Path, "/events/submissions/"), "%d", &id); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprintf(w, "event: submission.completed\ndata: {\"submission_id\":%d,\"attempt_id\":\"attempt\",\"status\":\"ACCEPTED\"}\n\n", id)
		default:
			t.Fatalf("unexpected request %s %s", request.Method, request.URL.Path)
		}
	}))
	defer server.Close()
	source := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(source, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	users := writeCredentialPool(t, []credentials.User{{Alias: "bench-001", Cookies: credentials.Cookies{AccessToken: firstToken}}, {Alias: "bench-002", Cookies: credentials.Cookies{AccessToken: secondToken}}})
	cfg := baseConfig(t, server.URL, users, source)
	cfg.Mode = config.ModeSustained
	cfg.Rate = big.NewRat(10, 1)
	cfg.RateRaw = "10"
	cfg.Duration = 0
	cfg.TotalSubmissions = 2
	cfg.WarmupCount = 1
	cfg.MaxSubmissions = 3
	cfg.CooldownGuard = 0
	cfg.SubmitLatencyBudget = 0
	cfg.PoolHeadroomPercent = 0
	prepared, err := Preflight(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), prepared)
	if err != nil {
		t.Fatal(err)
	}
	if run.Summary.Counts.Intended != 2 || posts.Load() != 3 {
		t.Fatalf("intended=%d posts=%d", run.Summary.Counts.Intended, posts.Load())
	}
	var metadata model.RunMetadata
	content, err := os.ReadFile(filepath.Join(run.Dir, "run.json"))
	if err != nil || json.Unmarshal(content, &metadata) != nil {
		t.Fatalf("read/decode run metadata: %v", err)
	}
	if metadata.Workload.TotalSubmissions == nil || *metadata.Workload.TotalSubmissions != 2 || metadata.Workload.ArrivalDurationMS != nil {
		t.Fatalf("workload=%+v", metadata.Workload)
	}
	reportText, err := os.ReadFile(filepath.Join(run.Dir, "report.md"))
	if err != nil || !strings.Contains(string(reportText), "Measured volume (intended): 2 submissions") {
		t.Fatalf("report=%q err=%v", reportText, err)
	}
}

func TestExactVolumeSchedulingStopsPromptlyOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cfg := config.Defaults(config.ModeSustained)
	cfg.Rate = big.NewRat(1, 1)
	cfg.TotalSubmissions = 10
	prepared := &Prepared{Config: cfg}
	run := newRun(ctx, prepared, nil)
	run.sustained()
	if len(run.records) != 1 || run.records[0].Outcome != model.OutcomeCancelled {
		t.Fatalf("records=%+v", run.records)
	}
}

func TestBurstMetricsUseActualTimestampIntervals(t *testing.T) {
	origin := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	firstStart, secondStart := origin.Add(10*time.Millisecond), origin.Add(30*time.Millisecond)
	firstAccepted, secondAccepted := origin.Add(20*time.Millisecond), origin.Add(60*time.Millisecond)
	firstTerminal, secondTerminal := origin.Add(time.Second), origin.Add(3*time.Second)
	launch := int64(35)
	metrics := burstMetrics([]model.SubmissionRecord{
		{Phase: model.PhaseLoad, PostStartedAt: &firstStart, Accepted: true, PostCompletedAt: &firstAccepted, TerminalObservedAt: &firstTerminal},
		{Phase: model.PhaseLoad, PostStartedAt: &secondStart, Accepted: true, PostCompletedAt: &secondAccepted, TerminalObservedAt: &secondTerminal},
	}, origin, &launch, 2, 2)
	if metrics.AttemptedIntervalMS == nil || *metrics.AttemptedIntervalMS != 20 || metrics.AcceptedIntervalMS == nil || *metrics.AcceptedIntervalMS != 40 || metrics.TerminalIntervalMS == nil || *metrics.TerminalIntervalMS != 2000 || metrics.AttemptedThroughputPerSec == nil || metrics.AcceptedThroughputPerSec == nil || metrics.TerminalThroughputPerSec == nil || metrics.PostStartOffsetMS.P99 == nil || *metrics.PostStartOffsetMS.P99 != 30 {
		t.Fatalf("metrics=%+v", metrics)
	}
	if metrics.PipelineTerminalThroughputPerSec == nil || *metrics.PipelineTerminalThroughputPerSec != *metrics.TerminalThroughputPerSec || !strings.Contains(metrics.TerminalThroughputSemantics, "pipeline") {
		t.Fatalf("canonical pipeline terminal metrics=%+v", metrics)
	}
	if single := burstMetrics([]model.SubmissionRecord{{Phase: model.PhaseLoad, PostStartedAt: &firstStart}}, origin, nil, 1, 0); single.AttemptedIntervalMS != nil || single.AttemptedThroughputPerSec != nil {
		t.Fatalf("single event fabricated throughput: %+v", single)
	}
}

func TestSummaryDoesNotInferJudgeCoreFromPipelineTerminalObservations(t *testing.T) {
	start := time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC)
	accepted, terminal := start.Add(10*time.Millisecond), start.Add(time.Second)
	records := []model.SubmissionRecord{{Phase: model.PhaseLoad, Accepted: true, PostCompletedAt: &accepted, TerminalObservedAt: &terminal, TerminalStatus: "ACCEPTED"}, {Phase: model.PhaseLoad, Accepted: true, PostCompletedAt: &accepted}}
	summary := summarize(config.Defaults(config.ModeSustained), records, nil, start, start.Add(time.Second), start, start.Add(2*time.Second), map[string]struct{}{}, 1)
	if summary.Compile.IncludedInJudgeCore || summary.JudgeCore.Availability != "UNAVAILABLE" || summary.JudgeCore.ThroughputPerSec != nil || summary.JudgeCore.WallMS != nil {
		t.Fatalf("Judge Core was inferred from pipeline data: %+v", summary)
	}
	if summary.Pipeline.TerminalCompleted != 1 || !summary.Pipeline.RightCensored || summary.Pipeline.TerminalObservationCoverage == nil || *summary.Pipeline.TerminalObservationCoverage != 0.5 {
		t.Fatalf("pipeline coverage=%+v", summary.Pipeline)
	}
	if !strings.Contains(summary.Pipeline.TerminalThroughputSemantics, "not Judge Core") {
		t.Fatalf("pipeline semantics=%q", summary.Pipeline.TerminalThroughputSemantics)
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

func TestSummarizeAggregatesLoadObserverTotals(t *testing.T) {
	start := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Minute)
	records := make([]model.SubmissionRecord, 0, 6)
	for index := 0; index < 4; index++ {
		records = append(records, model.SubmissionRecord{Phase: model.PhaseLoad, TerminalStatus: "ACCEPTED", CompletionSource: model.CompletionSSEEvent, GETReconciliations: 1, SSEFailures: index % 2})
	}
	records = append(records,
		model.SubmissionRecord{Phase: model.PhaseLoad, TerminalStatus: "ACCEPTED", CompletionSource: model.CompletionSSESnapshot, GETReconciliations: 2, SSEFailures: 1},
		model.SubmissionRecord{Phase: model.PhaseLoad, TerminalStatus: "ACCEPTED", CompletionSource: model.CompletionGETReconcile, GETReconciliations: 3, SSEFailures: 2},
		model.SubmissionRecord{Phase: model.PhaseLoad, CompletionSource: model.CompletionSSEEvent},
		model.SubmissionRecord{Phase: model.PhaseWarmup, TerminalStatus: "ACCEPTED", CompletionSource: model.CompletionSSEEvent, GETReconciliations: 99, SSEFailures: 99},
	)
	summary := summarize(config.Defaults(config.ModeBurst), records, nil, start, end, end, end, map[string]struct{}{}, 0)
	if summary.Observer.SSECompletions != 5 || summary.Observer.GETReconciliations != 9 || summary.Observer.SSEFailures != 5 {
		t.Fatalf("observer=%+v", summary.Observer)
	}
}

func TestPreflightSessionRefreshLifecycle(t *testing.T) {
	longLived := jwtExp(time.Now().Add(2 * time.Hour))
	shortLived := jwtExp(time.Now().Add(10 * time.Second))
	expired := jwtExp(time.Now().Add(-time.Minute))
	for _, test := range []struct {
		name        string
		access      string
		refresh     string
		mode        config.RefreshMode
		refreshed   string
		wantErr     string
		wantRefresh int32
		wantMe      int32
	}{
		{name: "auto keeps sufficient access", access: longLived, mode: config.RefreshAuto, refreshed: longLived, wantMe: 1},
		{name: "auto refreshes expired access", access: expired, refresh: "refresh-secret", mode: config.RefreshAuto, refreshed: longLived, wantRefresh: 1, wantMe: 1},
		{name: "auto refreshes insufficient horizon", access: shortLived, refresh: "refresh-secret", mode: config.RefreshAuto, refreshed: longLived, wantRefresh: 1, wantMe: 1},
		{name: "auto expired without refresh fails clearly", access: expired, mode: config.RefreshAuto, refreshed: longLived, wantErr: "no refresh token", wantMe: 0},
		{name: "refresh off never refreshes", access: expired, refresh: "refresh-secret", mode: config.RefreshOff, refreshed: longLived, wantErr: "--session-refresh=off", wantMe: 0},
		{name: "refresh required refreshes", access: longLived, refresh: "refresh-secret", mode: config.RefreshRequired, refreshed: longLived, wantRefresh: 1, wantMe: 1},
		{name: "refresh required needs token", access: longLived, mode: config.RefreshRequired, refreshed: longLived, wantErr: "refresh is required", wantMe: 0},
		{name: "short refreshed token fails", access: expired, refresh: "refresh-secret", mode: config.RefreshAuto, refreshed: shortLived, wantErr: "after refresh", wantRefresh: 1, wantMe: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			var refreshes, meCalls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				switch request.URL.Path {
				case "/api/v1/auth/refresh-token":
					refreshes.Add(1)
					_, refreshCookieErr := request.Cookie("refresh_token")
					if request.Method != http.MethodPost || refreshCookieErr != nil {
						t.Fatalf("invalid refresh request")
					}
					http.SetCookie(w, &http.Cookie{Name: "access_token", Value: test.refreshed, Path: "/", HttpOnly: true})
					_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":null}`))
				case "/api/v1/me":
					meCalls.Add(1)
					_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":{"id":"real-user-id","role":"user","is_active":true}}`))
				case "/api/v1/problems/sample":
					_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":{"id":7}}`))
				default:
					t.Fatalf("unexpected request %s %s", request.Method, request.URL.Path)
				}
			}))
			defer server.Close()
			source := filepath.Join(t.TempDir(), "main.go")
			if err := os.WriteFile(source, []byte("source"), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg := baseConfig(t, server.URL, writeCredentialPool(t, []credentials.User{{Alias: "bench-001", Cookies: credentials.Cookies{AccessToken: test.access, RefreshToken: test.refresh}}}), source)
			cfg.RefreshMode = test.mode
			prepared, err := Preflight(context.Background(), cfg)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) || strings.Contains(err.Error(), "refresh-secret") {
					t.Fatalf("err=%v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			} else if prepared.Subjects["bench-001"] != "real-user-id" || prepared.Sessions[0].Subject != "real-user-id" {
				t.Fatalf("refreshed identity was not validated: %+v", prepared)
			}
			if refreshes.Load() != test.wantRefresh || meCalls.Load() != test.wantMe {
				t.Fatalf("refreshes=%d me=%d, want %d/%d", refreshes.Load(), meCalls.Load(), test.wantRefresh, test.wantMe)
			}
		})
	}
}

func TestPreflightRejectsDuplicateAuthenticatedIdentitiesAfterSessionValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/me" {
			t.Fatalf("unexpected request %s", request.URL.Path)
		}
		_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":{"id":"same-user","role":"user","is_active":true}}`))
	}))
	defer server.Close()
	source := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(source, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	users := []credentials.User{
		{Alias: "bench-001", Cookies: credentials.Cookies{AccessToken: jwtExp(time.Now().Add(2 * time.Hour))}},
		{Alias: "bench-002", Cookies: credentials.Cookies{AccessToken: jwtExp(time.Now().Add(3 * time.Hour))}},
	}
	_, err := Preflight(context.Background(), baseConfig(t, server.URL, writeCredentialPool(t, users), source))
	if err == nil || !strings.Contains(err.Error(), "duplicate authenticated identities") {
		t.Fatalf("err=%v", err)
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
