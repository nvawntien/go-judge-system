// Package model contains result-only benchmark records. These types deliberately
// exclude credentials, source text, ticket values, and authenticated user IDs.
package model

import "time"

const BenchmarkVersion = "judge-bench/v1"

type Phase string

const (
	PhaseWarmup Phase = "warmup"
	PhaseLoad   Phase = "load"
	PhaseDrain  Phase = "drain"
)

type Mode string

const (
	ModeBurst     Mode = "burst"
	ModeSustained Mode = "sustained"
)

type RunState string

const (
	RunCompleted RunState = "completed"
	RunAborted   RunState = "aborted"
	RunCancelled RunState = "cancelled"
)

type Classification string

const (
	ClassificationStable       Classification = "STABLE"
	ClassificationSaturated    Classification = "SATURATED"
	ClassificationInconclusive Classification = "INCONCLUSIVE"
	ClassificationNA           Classification = "N/A"
)

type Outcome string

const (
	OutcomeTerminal          Outcome = "terminal"
	OutcomeRejected429       Outcome = "rejected_429"
	OutcomeRejected4xx       Outcome = "other_4xx"
	OutcomeServerError       Outcome = "server_error"
	OutcomeAmbiguousPost     Outcome = "ambiguous_post"
	OutcomeAuthFailure       Outcome = "auth_failure"
	OutcomeUserExhausted     Outcome = "scheduler_user_exhausted"
	OutcomeLocalFailure      Outcome = "local_request_failure"
	OutcomeCompletionTimeout Outcome = "completion_timeout"
	OutcomeObserverFailure   Outcome = "observer_failure"
	OutcomeCancelled         Outcome = "cancelled"
)

type CompletionSource string

const (
	CompletionSSESnapshot  CompletionSource = "sse_snapshot"
	CompletionSSEEvent     CompletionSource = "sse_event"
	CompletionGETReconcile CompletionSource = "get_reconcile"
	CompletionGETSafety    CompletionSource = "get_safety"
	CompletionGETFallback  CompletionSource = "get_fallback"
)

type RunMetadata struct {
	SchemaVersion    string         `json:"schema_version"`
	BenchmarkVersion string         `json:"benchmark_version"`
	RunID            string         `json:"run_id"`
	Mode             Mode           `json:"mode"`
	Repetition       int            `json:"repetition"`
	Seed             int64          `json:"seed"`
	State            RunState       `json:"state"`
	StartedAt        time.Time      `json:"started_at"`
	EndedAt          *time.Time     `json:"ended_at,omitempty"`
	Repository       Repository     `json:"repository"`
	Target           Target         `json:"target"`
	Users            UserSet        `json:"users"`
	Workload         Workload       `json:"workload"`
	Timeouts         Timeouts       `json:"timeouts"`
	Observer         ObserverConfig `json:"observer"`
	Phases           Phases         `json:"phases"`
	ObservedRates    *ObservedRates `json:"observed_rates,omitempty"`
}

type Repository struct {
	GitSHA string `json:"git_sha"`
	Dirty  bool   `json:"dirty"`
}

type Target struct {
	BaseURL         string `json:"base_url"`
	ProblemID       int64  `json:"problem_id"`
	ProblemSlug     string `json:"problem_slug"`
	Language        string `json:"language"`
	ExpectedVerdict string `json:"expected_verdict"`
	SourceSHA256    string `json:"source_sha256"`
	SourceBytes     int64  `json:"source_bytes"`
}

type UserSet struct {
	Configured int `json:"configured"`
	Selected   int `json:"selected"`
}

type Workload struct {
	BurstSize             *int     `json:"burst_size,omitempty"`
	JitterMilliseconds    *int64   `json:"jitter_ms,omitempty"`
	TargetRatePerSecond   *float64 `json:"target_rate_per_second,omitempty"`
	ArrivalDurationMS     *int64   `json:"arrival_duration_ms,omitempty"`
	WindowMS              int64    `json:"window_ms"`
	WarmupCount           int      `json:"warmup_count"`
	SubmitCooldownMS      int64    `json:"submit_cooldown_ms"`
	CooldownGuardMS       int64    `json:"cooldown_guard_ms"`
	SubmitLatencyBudgetMS int64    `json:"submit_latency_budget_ms"`
	PoolHeadroomPercent   float64  `json:"pool_headroom_percent"`
	MaxSubmissions        int      `json:"max_submissions"`
	MaxInFlight           int      `json:"max_in_flight"`
}

type Timeouts struct {
	APIMS        int64 `json:"api_ms"`
	SubmissionMS int64 `json:"submission_ms"`
	DrainMS      int64 `json:"drain_ms"`
}

type ObserverConfig struct {
	SSEPrimary                bool    `json:"sse_primary"`
	ConnectMS                 int64   `json:"connect_ms"`
	IdleMS                    int64   `json:"idle_ms"`
	MaxReconnects             int     `json:"max_reconnects"`
	BackoffBaseMS             int64   `json:"backoff_base_ms"`
	BackoffMaxMS              int64   `json:"backoff_max_ms"`
	SafetyReconcileIntervalMS int64   `json:"safety_reconcile_interval_ms"`
	ReconcileMaxQPS           float64 `json:"reconcile_max_qps"`
}

type Phases struct {
	Warmup PhaseTiming `json:"warmup"`
	Load   PhaseTiming `json:"load"`
	Drain  PhaseTiming `json:"drain"`
}

type PhaseTiming struct {
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
}

type ObservedRates struct {
	AttemptedPerSecond float64 `json:"attempted_per_second"`
	AcceptedPerSecond  float64 `json:"accepted_per_second"`
	CompletedPerSecond float64 `json:"completed_per_second"`
}

// SubmissionRecord has only safe aliases and API-visible identifiers. It never
// carries a source program, ticket, cookie, token, or real authenticated ID.
type SubmissionRecord struct {
	RunID                 string           `json:"run_id"`
	Phase                 Phase            `json:"phase"`
	Sequence              int              `json:"sequence"`
	UserAlias             string           `json:"user_alias"`
	IntendedAt            *time.Time       `json:"intended_at,omitempty"`
	IntendedOffsetMS      *int64           `json:"intended_offset_ms,omitempty"`
	PostStartedAt         *time.Time       `json:"post_started_at,omitempty"`
	PostStartOffsetMS     *int64           `json:"post_start_offset_ms,omitempty"`
	PostCompletedAt       *time.Time       `json:"post_completed_at,omitempty"`
	PostCompletedOffsetMS *int64           `json:"post_completed_offset_ms,omitempty"`
	ScheduleDelayMS       *float64         `json:"schedule_delay_ms,omitempty"`
	SubmitLatencyMS       *float64         `json:"submit_latency_ms,omitempty"`
	Attempted             bool             `json:"attempted"`
	Accepted              bool             `json:"accepted"`
	HTTPStatus            *int             `json:"http_status,omitempty"`
	APICode               *int             `json:"api_code,omitempty"`
	SubmissionID          *int64           `json:"submission_id,omitempty"`
	InitialStatus         string           `json:"initial_status,omitempty"`
	TerminalStatus        string           `json:"terminal_status,omitempty"`
	TerminalObservedAt    *time.Time       `json:"terminal_observed_at,omitempty"`
	TerminalOffsetMS      *int64           `json:"terminal_offset_ms,omitempty"`
	EndToEndLatencyMS     *float64         `json:"end_to_end_latency_ms,omitempty"`
	AcceptedToTerminalMS  *float64         `json:"accepted_to_terminal_ms,omitempty"`
	ObserverStartedAt     *time.Time       `json:"observer_started_at,omitempty"`
	ObserverEndedAt       *time.Time       `json:"observer_ended_at,omitempty"`
	CompletionSource      CompletionSource `json:"completion_source,omitempty"`
	SSEConnections        int              `json:"sse_connections"`
	SSEFailures           int              `json:"sse_failures"`
	GETReconciliations    int              `json:"get_reconciliations"`
	RateLimited           bool             `json:"rate_limited"`
	RetryAfterMS          *int64           `json:"retry_after_ms,omitempty"`
	Outcome               Outcome          `json:"outcome"`
	ErrorClass            string           `json:"error_class,omitempty"`
	Error                 string           `json:"error,omitempty"`
}

type WindowRecord struct {
	RunID                   string    `json:"run_id"`
	Phase                   Phase     `json:"phase"`
	WindowIndex             int       `json:"window_index"`
	WindowStart             time.Time `json:"window_start"`
	WindowEnd               time.Time `json:"window_end"`
	WindowStartOffsetMS     int64     `json:"window_start_offset_ms"`
	WindowEndOffsetMS       int64     `json:"window_end_offset_ms"`
	WindowDurationMS        int64     `json:"window_duration_ms"`
	Intended                int       `json:"intended"`
	Attempted               int       `json:"attempted"`
	Accepted                int       `json:"accepted"`
	Completed               int       `json:"completed"`
	IntendedCumulative      int       `json:"intended_cumulative"`
	AttemptedCumulative     int       `json:"attempted_cumulative"`
	AcceptedCumulative      int       `json:"accepted_cumulative"`
	CompletedCumulative     int       `json:"completed_cumulative"`
	ClientOutstandingStart  int       `json:"client_outstanding_start"`
	ClientOutstanding       int       `json:"client_outstanding"`
	ClientOutstandingPeak   int       `json:"client_outstanding_peak"`
	TargetRatePerSecond     *float64  `json:"target_arrival_rate_per_sec,omitempty"`
	AttemptedRatePerSecond  float64   `json:"attempted_rate_per_sec"`
	AcceptedRatePerSecond   float64   `json:"accepted_rate_per_sec"`
	CompletionRatePerSecond float64   `json:"completion_rate_per_sec"`
	ScheduleDelayP50MS      *float64  `json:"schedule_delay_p50_ms,omitempty"`
	ScheduleDelayP95MS      *float64  `json:"schedule_delay_p95_ms,omitempty"`
	ScheduleDelayP99MS      *float64  `json:"schedule_delay_p99_ms,omitempty"`
	ScheduleDelayMaxMS      *float64  `json:"schedule_delay_max_ms,omitempty"`
	SubmitLatencyP50MS      *float64  `json:"submit_latency_p50_ms,omitempty"`
	SubmitLatencyP95MS      *float64  `json:"submit_latency_p95_ms,omitempty"`
	E2ELatencyP50MS         *float64  `json:"e2e_latency_p50_ms,omitempty"`
	E2ELatencyP95MS         *float64  `json:"e2e_latency_p95_ms,omitempty"`
	RateLimited             int       `json:"rate_limited"`
	Other4xx                int       `json:"other_4xx"`
	ServerErrors            int       `json:"server_errors"`
	TransportFailures       int       `json:"transport_failures"`
	CompletionTimeouts      int       `json:"completion_timeouts"`
	ActiveObserversEnd      int       `json:"active_observers_end"`
	PeakInFlight            int       `json:"peak_in_flight"`
}

type Distribution struct {
	Count int      `json:"count"`
	Min   *float64 `json:"min,omitempty"`
	Mean  *float64 `json:"mean,omitempty"`
	P50   *float64 `json:"p50,omitempty"`
	P95   *float64 `json:"p95,omitempty"`
	P99   *float64 `json:"p99,omitempty"`
	Max   *float64 `json:"max,omitempty"`
}

type RunSummary struct {
	SchemaVersion         string          `json:"schema_version"`
	RunID                 string          `json:"run_id"`
	RunState              RunState        `json:"run_state"`
	Classification        Classification  `json:"classification"`
	ClassificationReasons []string        `json:"classification_reasons"`
	QualityFlags          []string        `json:"quality_flags"`
	Counts                Counts          `json:"counts"`
	Rates                 Rates           `json:"rates"`
	LoadWindow            LoadWindow      `json:"load_window"`
	Drain                 Drain           `json:"drain"`
	Outstanding           Outstanding     `json:"outstanding"`
	Latencies             Latencies       `json:"latencies"`
	Verdicts              map[string]int  `json:"verdicts"`
	Observer              ObserverTotals  `json:"observer"`
	ExternalMetrics       ExternalMetrics `json:"external_metrics"`
}

type Counts struct {
	Intended            int `json:"intended"`
	Attempted           int `json:"attempted"`
	Accepted            int `json:"accepted"`
	Terminal            int `json:"terminal"`
	RateLimited         int `json:"rate_limited"`
	Other4xx            int `json:"other_4xx"`
	ServerErrors        int `json:"server_errors"`
	TransportFailures   int `json:"transport_failures"`
	AmbiguousPosts      int `json:"ambiguous_posts"`
	CompletionTimeouts  int `json:"completion_timeouts"`
	UserPoolExhaustions int `json:"user_pool_exhaustions"`
}

type Rates struct {
	TargetArrivalPerSecond       *float64 `json:"target_arrival_per_second,omitempty"`
	LoadAttemptedPerSecond       float64  `json:"load_attempted_per_second"`
	LoadAcceptedPerSecond        float64  `json:"load_accepted_per_second"`
	LoadTerminalCompletionSecond float64  `json:"load_terminal_completion_per_second"`
}

type LoadWindow struct {
	DurationMS                int64  `json:"duration_ms"`
	Accepted                  int    `json:"accepted"`
	BoundaryAcceptedAfterLoad int    `json:"boundary_accepted_after_load"`
	Completed                 int    `json:"completed"`
	OutstandingAtEnd          int    `json:"outstanding_at_end"`
	PostsInFlightAtEnd        int    `json:"posts_in_flight_at_end"`
	BurstSpreadMS             *int64 `json:"burst_spread_ms,omitempty"`
}

type Drain struct {
	DurationMS              int64    `json:"duration_ms"`
	OutstandingAtStart      int      `json:"outstanding_at_start"`
	Completed               int      `json:"completed"`
	Remaining               int      `json:"remaining"`
	CompletionRatePerSecond *float64 `json:"completion_rate_per_second,omitempty"`
	TimedOut                bool     `json:"timed_out"`
}

type Outstanding struct {
	Peak           int      `json:"peak"`
	EndOfLoad      int      `json:"end_of_load"`
	EndOfDrain     int      `json:"end_of_drain"`
	SlopePerMinute *float64 `json:"window_slope_per_minute,omitempty"`
}

type Latencies struct {
	SubmitMS        Distribution `json:"submit_ms"`
	EndToEndMS      Distribution `json:"end_to_end_ms"`
	ScheduleDelayMS Distribution `json:"schedule_delay_ms"`
}

type ObserverTotals struct {
	SSECompletions     int `json:"sse_completions"`
	GETReconciliations int `json:"get_reconciliations"`
	SSEFailures        int `json:"sse_failures"`
}

type ExternalMetrics struct {
	AppStats   *ExternalInput `json:"app_stats,omitempty"`
	JudgeStats *ExternalInput `json:"judge_stats,omitempty"`
	KafkaLag   *ExternalInput `json:"kafka_lag,omitempty"`
}

type ExternalInput struct {
	Status  string `json:"status"`
	SHA256  string `json:"sha256,omitempty"`
	Samples int    `json:"samples,omitempty"`
}
