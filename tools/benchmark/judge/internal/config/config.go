// Package config parses and validates the benchmark's intentionally explicit
// workload and safety controls. It never discovers or changes server settings.
package config

import (
	"errors"
	"flag"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/model"
)

type Mode string

const (
	ModeBurst     Mode = "burst"
	ModeSustained Mode = "sustained"
)

type RefreshMode string

const (
	RefreshAuto     RefreshMode = "auto"
	RefreshRequired RefreshMode = "required"
	RefreshOff      RefreshMode = "off"
)

type ErrorPolicy string

const (
	ErrorPolicyStop     ErrorPolicy = "stop"
	ErrorPolicyContinue ErrorPolicy = "continue"
)

type Objective string

const (
	ObjectiveJudgeCapacity    Objective = "judge-capacity"
	ObjectiveAdmissionControl Objective = "admission-control"
	ObjectiveMassiveBurst     Objective = "massive-burst"
)

// ObservationMode separates an intake-only burst from the existing
// SSE/reconciliation based terminal-observation benchmark.  It is deliberately
// explicit because the two modes measure different systems.
type ObservationMode string

const (
	ObservationFull          ObservationMode = "full"
	ObservationAdmissionOnly ObservationMode = "admission-only"
	// ObservationRealistic models the browser-facing submit -> ticket -> SSE
	// sequence without waiting for the Judge pipeline to drain.
	ObservationRealistic ObservationMode = "realistic"
)

type Config struct {
	Mode                     Mode
	BaseURL                  *url.URL
	BaseURLRaw               string
	UsersFile                string
	ProblemID                int64
	ProblemSlug              string
	Language                 string
	SourceFile               string
	ExpectedVerdict          string
	ResultRoot               string
	RunID                    string
	Repetition               int
	Seed                     int64
	WarmupCount              int
	WarmupTimeout            time.Duration
	SubmitCooldown           time.Duration
	CooldownGuard            time.Duration
	SubmitLatencyBudget      time.Duration
	PoolHeadroomPercent      float64
	APITimeout               time.Duration
	SubmissionTimeout        time.Duration
	DrainTimeout             time.Duration
	Window                   time.Duration
	MaxSubmissions           int
	MaxInFlight              int
	UserCount                int
	PreflightConcurrency     int
	BurstStartTimeout        time.Duration
	ErrorPolicy              ErrorPolicy
	Objective                Objective
	ObservationMode          ObservationMode
	AdmissionPreflightSample int
	SSEHoldDuration          time.Duration
	SSEConnectTimeout        time.Duration
	SSEIdleTimeout           time.Duration
	SSEMaxReconnects         int
	SSEBackoffBase           time.Duration
	SSEBackoffMax            time.Duration
	SafetyReconcileInterval  time.Duration
	ReconcileMaxQPS          float64
	RefreshMode              RefreshMode
	AuthValidityMargin       time.Duration
	AllowRemote              bool
	ConfirmTargetHost        string
	BurstSize                int
	Jitter                   time.Duration
	ScheduleDelayP95Budget   time.Duration
	Rate                     *big.Rat
	RateRaw                  string
	Duration                 time.Duration
	TotalSubmissions         int
	SystemConfigFile         string
	SystemConfig             *model.SystemConfig
}

func Defaults(mode Mode) Config {
	return Config{
		Mode:                mode,
		ResultRoot:          "bench-results",
		Repetition:          1,
		WarmupTimeout:       2 * time.Minute,
		CooldownGuard:       100 * time.Millisecond,
		SubmitLatencyBudget: 500 * time.Millisecond,
		PoolHeadroomPercent: 25,
		APITimeout:          5 * time.Second,
		SubmissionTimeout:   5 * time.Minute,
		DrainTimeout:        5 * time.Minute,
		Window:              30 * time.Second,
		MaxInFlight:         100,
		// Session validation is bounded, but a 100k pool must complete its
		// authenticated preflight before short-lived access sessions age out.
		PreflightConcurrency:     256,
		BurstStartTimeout:        2 * time.Minute,
		ErrorPolicy:              ErrorPolicyStop,
		Objective:                ObjectiveJudgeCapacity,
		ObservationMode:          ObservationFull,
		AdmissionPreflightSample: 32,
		SSEHoldDuration:          30 * time.Second,
		SSEConnectTimeout:        10 * time.Second,
		SSEIdleTimeout:           45 * time.Second,
		SSEMaxReconnects:         5,
		SSEBackoffBase:           time.Second,
		SSEBackoffMax:            10 * time.Second,
		SafetyReconcileInterval:  60 * time.Second,
		ReconcileMaxQPS:          2,
		RefreshMode:              RefreshAuto,
		AuthValidityMargin:       time.Minute,
		ScheduleDelayP95Budget:   25 * time.Millisecond,
	}
}

// Bind flags shared by preflight, burst, and sustained. The caller selects the
// mode and invokes Validate after flag parsing.
func Bind(fs *flag.FlagSet, cfg *Config) {
	fs.Var(urlValue{cfg: cfg}, "base-url", "gateway base URL")
	fs.StringVar(&cfg.UsersFile, "users-file", "", "local benchmark session file")
	fs.Int64Var(&cfg.ProblemID, "problem-id", 0, "canonical public problem ID")
	fs.StringVar(&cfg.ProblemSlug, "problem-slug", "", "public problem slug")
	fs.StringVar(&cfg.Language, "language", "", "submission language")
	fs.StringVar(&cfg.SourceFile, "source-file", "", "source file path")
	fs.StringVar(&cfg.ExpectedVerdict, "expected-verdict", "", "required terminal verdict")
	fs.DurationVar(&cfg.SubmitCooldown, "submit-cooldown", 0, "deployed per-user/problem cooldown")
	fs.StringVar(&cfg.ResultRoot, "result-root", cfg.ResultRoot, "result root directory")
	fs.StringVar(&cfg.RunID, "run-id", "", "stable run ID")
	fs.IntVar(&cfg.Repetition, "repetition", cfg.Repetition, "repetition number")
	fs.Int64Var(&cfg.Seed, "seed", 0, "deterministic seed; zero generates one")
	fs.IntVar(&cfg.WarmupCount, "warmup-count", 0, "warmup submissions")
	fs.DurationVar(&cfg.WarmupTimeout, "warmup-timeout", cfg.WarmupTimeout, "warmup deadline")
	fs.DurationVar(&cfg.CooldownGuard, "cooldown-guard", cfg.CooldownGuard, "post-cooldown safety guard")
	fs.DurationVar(&cfg.SubmitLatencyBudget, "submit-latency-budget", cfg.SubmitLatencyBudget, "pool sizing submit latency budget")
	fs.Float64Var(&cfg.PoolHeadroomPercent, "pool-headroom-percent", cfg.PoolHeadroomPercent, "extra user-pool headroom percent")
	fs.DurationVar(&cfg.APITimeout, "api-timeout", cfg.APITimeout, "ordinary API request timeout")
	fs.DurationVar(&cfg.SubmissionTimeout, "submission-timeout", cfg.SubmissionTimeout, "accepted submission completion deadline")
	fs.DurationVar(&cfg.DrainTimeout, "drain-timeout", cfg.DrainTimeout, "post-load drain deadline")
	fs.DurationVar(&cfg.Window, "window", cfg.Window, "statistics window duration")
	fs.IntVar(&cfg.MaxSubmissions, "max-submissions", 0, "hard cap for all submissions including warmup")
	fs.IntVar(&cfg.MaxInFlight, "max-in-flight", cfg.MaxInFlight, "logical submission cap; unresolved accepted work in full/realistic observation, active POSTs in admission-only")
	fs.IntVar(&cfg.UserCount, "user-count", 0, "select first N canonical benchmark sessions; required for massive burst")
	fs.IntVar(&cfg.PreflightConcurrency, "preflight-concurrency", cfg.PreflightConcurrency, "bounded concurrent session validation/refresh work")
	fs.DurationVar(&cfg.BurstStartTimeout, "burst-start-timeout", cfg.BurstStartTimeout, "burst launch and event-ticket acquisition lifetime horizon")
	fs.Var(enumValue{get: func() string { return string(cfg.ErrorPolicy) }, set: func(value string) { cfg.ErrorPolicy = ErrorPolicy(value) }}, "submit-error-policy", "stop or continue")
	fs.Var(enumValue{get: func() string { return string(cfg.Objective) }, set: func(value string) { cfg.Objective = Objective(value) }}, "benchmark-objective", "judge-capacity, admission-control, or massive-burst")
	fs.Var(enumValue{get: func() string { return string(cfg.ObservationMode) }, set: func(value string) { cfg.ObservationMode = ObservationMode(value) }}, "observation-mode", "full, realistic, or admission-only; realistic uses one ticket and one bounded SSE attempt per accepted POST")
	fs.IntVar(&cfg.AdmissionPreflightSample, "admission-preflight-sample", cfg.AdmissionPreflightSample, "admission-only deterministic session sample; first and last identities are always checked")
	fs.DurationVar(&cfg.SSEHoldDuration, "sse-hold-duration", cfg.SSEHoldDuration, "realistic mode: retain an established SSE stream for this duration unless a terminal event arrives")
	fs.DurationVar(&cfg.SSEConnectTimeout, "sse-connect-timeout", cfg.SSEConnectTimeout, "SSE connection timeout")
	fs.DurationVar(&cfg.SSEIdleTimeout, "sse-idle-timeout", cfg.SSEIdleTimeout, "SSE idle timeout")
	fs.IntVar(&cfg.SSEMaxReconnects, "sse-max-reconnects", cfg.SSEMaxReconnects, "bounded SSE reconnects")
	fs.DurationVar(&cfg.SSEBackoffBase, "sse-backoff-base", cfg.SSEBackoffBase, "SSE reconnect backoff base")
	fs.DurationVar(&cfg.SSEBackoffMax, "sse-backoff-max", cfg.SSEBackoffMax, "SSE reconnect backoff cap")
	fs.DurationVar(&cfg.SafetyReconcileInterval, "safety-reconcile-interval", cfg.SafetyReconcileInterval, "conservative SSE safety GET cadence; zero disables")
	fs.Float64Var(&cfg.ReconcileMaxQPS, "reconcile-max-qps", cfg.ReconcileMaxQPS, "global GET reconciliation cap")
	fs.Var(enumValue{get: func() string { return string(cfg.RefreshMode) }, set: func(value string) { cfg.RefreshMode = RefreshMode(value) }}, "session-refresh", "auto, required, or off")
	fs.DurationVar(&cfg.AuthValidityMargin, "auth-validity-margin", cfg.AuthValidityMargin, "token lifetime safety margin")
	fs.BoolVar(&cfg.AllowRemote, "allow-remote", false, "allow a confirmed non-loopback HTTPS target")
	fs.StringVar(&cfg.ConfirmTargetHost, "confirm-target-host", "", "exact non-loopback target hostname")
	fs.IntVar(&cfg.BurstSize, "burst-size", 0, "distinct simultaneous burst users")
	fs.DurationVar(&cfg.Jitter, "jitter", 0, "deterministic uniform burst jitter")
	fs.DurationVar(&cfg.ScheduleDelayP95Budget, "schedule-delay-p95-budget", cfg.ScheduleDelayP95Budget, "burst p95 scheduling budget")
	fs.StringVar(&cfg.RateRaw, "rate", "", "sustained constant arrival rate per second")
	fs.DurationVar(&cfg.Duration, "duration", 0, "sustained arrival duration")
	fs.IntVar(&cfg.TotalSubmissions, "total-submissions", 0, "exact measured submissions in sustained mode; excludes warmup")
	fs.StringVar(&cfg.SystemConfigFile, "system-config", "", "safe system-under-test JSON metadata")

}

// SetBaseURL parses a raw URL supplied by the command layer.
func (c *Config) SetBaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse --base-url: %w", err)
	}
	c.BaseURL = u
	c.BaseURLRaw = raw
	return nil
}

// ParseRate accepts decimal text and preserves it as an exact rational.
func (c *Config) ParseRate(raw string) error {
	if raw == "" {
		c.Rate = nil
		c.RateRaw = ""
		return nil
	}
	r, ok := new(big.Rat).SetString(raw)
	if !ok || r.Sign() <= 0 {
		return fmt.Errorf("--rate must be a positive decimal")
	}
	c.Rate = r
	c.RateRaw = raw
	return nil
}

func (c Config) Validate() error {
	if c.BaseURL == nil {
		return errors.New("--base-url is required")
	}
	if c.BaseURL.Scheme != "http" && c.BaseURL.Scheme != "https" {
		return errors.New("--base-url must use http or https")
	}
	if c.BaseURL.Host == "" || c.BaseURL.User != nil || c.BaseURL.RawQuery != "" || c.BaseURL.Fragment != "" {
		return errors.New("--base-url must have a host and no userinfo, query, or fragment")
	}
	if c.BaseURL.Path != "" && c.BaseURL.Path != "/" {
		return errors.New("--base-url must not include a path")
	}
	if err := validateRequired(c); err != nil {
		return err
	}
	remote := !isLoopbackHost(c.BaseURL.Hostname())
	if remote {
		if c.BaseURL.Scheme != "https" {
			return errors.New("non-loopback targets require https")
		}
		if !c.AllowRemote || c.ConfirmTargetHost != c.BaseURL.Hostname() {
			return errors.New("non-loopback targets require --allow-remote and exact --confirm-target-host")
		}
		if c.MaxSubmissions <= 0 {
			return errors.New("non-loopback targets require --max-submissions")
		}
	}
	if c.TotalSubmissions < 0 {
		return errors.New("--total-submissions must not be negative")
	}
	if c.Mode == ModeBurst {
		if c.BurstSize <= 0 {
			return errors.New("--burst-size must be positive for burst mode")
		}
		if c.TotalSubmissions != 0 {
			return errors.New("--total-submissions is only valid for sustained mode")
		}
		if c.Objective == ObjectiveMassiveBurst {
			if c.UserCount <= 0 || c.UserCount != c.BurstSize {
				return errors.New("massive burst requires --user-count equal to --burst-size")
			}
			if c.WarmupCount != 0 {
				return errors.New("massive burst requires --warmup-count=0 to preserve one submission per user")
			}
			if c.Jitter != 0 {
				return errors.New("massive burst requires --jitter=0 for one common release origin")
			}
			if c.ErrorPolicy != ErrorPolicyContinue {
				return errors.New("massive burst requires --submit-error-policy=continue so every planned logical submission is released")
			}
			if c.MaxInFlight < c.BurstSize {
				return errors.New("massive burst requires --max-in-flight at least --burst-size")
			}
			// Full observation is SSE-first. A periodic GET for every one of
			// thousands of accepted submissions would turn a submission burst into
			// an artificial Gateway/Postgres read burst. Reconnect-triggered GET
			// reconciliation remains bounded by ReconcileMaxQPS; require the
			// operator to explicitly disable periodic safety reconciliation here.
			if c.SafetyReconcileInterval != 0 {
				return errors.New("massive burst requires --safety-reconcile-interval=0 to avoid a periodic reconciliation storm")
			}
		}
	}
	if c.ObservationMode == ObservationAdmissionOnly {
		if c.Mode != ModeBurst || c.Objective != ObjectiveMassiveBurst {
			return errors.New("--observation-mode=admission-only requires burst massive-burst mode")
		}
		if c.AdmissionPreflightSample < 0 || c.AdmissionPreflightSample > c.UserCount {
			return errors.New("--admission-preflight-sample must be within 0..--user-count")
		}
	}
	if c.ObservationMode == ObservationRealistic {
		if c.Mode != ModeBurst || c.Objective != ObjectiveMassiveBurst {
			return errors.New("--observation-mode=realistic requires burst massive-burst mode")
		}
		if c.SSEHoldDuration <= 0 {
			return errors.New("--sse-hold-duration must be positive for realistic observation")
		}
	}
	if c.Mode == ModeSustained {
		if c.Rate == nil {
			return errors.New("sustained mode requires positive --rate")
		}
		if c.Duration < 0 {
			return errors.New("--duration must not be negative")
		}
		if c.Duration > 0 && c.TotalSubmissions > 0 {
			return errors.New("sustained mode accepts either --duration or --total-submissions, not both")
		}
		if c.Duration == 0 && c.TotalSubmissions == 0 {
			return errors.New("sustained mode requires --duration or --total-submissions")
		}
	}
	if c.MaxSubmissions <= 0 {
		return errors.New("--max-submissions must be positive")
	}
	if planned := c.PlannedSubmissions(); planned > c.MaxSubmissions {
		return fmt.Errorf("planned submissions %d exceed --max-submissions %d", planned, c.MaxSubmissions)
	}
	if c.SafetyReconcileInterval > 0 {
		if c.ReconcileMaxQPS <= 0 {
			return errors.New("--reconcile-max-qps must be positive when safety reconciliation is enabled")
		}
		requiredQPS := float64(c.MaxInFlight) / c.SafetyReconcileInterval.Seconds()
		if requiredQPS > c.ReconcileMaxQPS {
			return fmt.Errorf("reconciliation capacity %.3f qps cannot cover max-in-flight %.0f within %s", c.ReconcileMaxQPS, float64(c.MaxInFlight), c.SafetyReconcileInterval)
		}
	}
	return nil
}

func validateRequired(c Config) error {
	if c.UsersFile == "" || c.ProblemID <= 0 || c.ProblemSlug == "" || c.SourceFile == "" || c.Language == "" || c.ExpectedVerdict == "" {
		return errors.New("--users-file, --problem-id, --problem-slug, --language, --source-file, and --expected-verdict are required")
	}
	if !validLanguage(c.Language) {
		return fmt.Errorf("unsupported --language %q", c.Language)
	}
	if c.SubmitCooldown <= 0 || c.CooldownGuard < 0 || c.SubmitLatencyBudget < 0 {
		return errors.New("cooldown values must be non-negative and --submit-cooldown must be positive")
	}
	if c.WarmupCount < 0 || c.Repetition <= 0 || c.MaxInFlight <= 0 || c.UserCount < 0 || c.UserCount > 100000 || c.PreflightConcurrency <= 0 || c.PreflightConcurrency > 512 || c.PoolHeadroomPercent < 0 {
		return errors.New("counts and pool headroom are invalid")
	}
	if c.APITimeout <= 0 || c.SubmissionTimeout <= 0 || c.DrainTimeout <= 0 || c.Window <= 0 || c.WarmupTimeout <= 0 || c.BurstStartTimeout <= 0 {
		return errors.New("timeouts and --window must be positive")
	}
	if c.SSEConnectTimeout <= 0 || c.SSEIdleTimeout <= 0 || c.SSEMaxReconnects < 0 || c.SSEBackoffBase <= 0 || c.SSEBackoffMax < c.SSEBackoffBase {
		return errors.New("SSE controls are invalid")
	}
	if c.RefreshMode != RefreshAuto && c.RefreshMode != RefreshRequired && c.RefreshMode != RefreshOff {
		return errors.New("--session-refresh must be auto, required, or off")
	}
	if c.ErrorPolicy != ErrorPolicyStop && c.ErrorPolicy != ErrorPolicyContinue {
		return errors.New("--submit-error-policy must be stop or continue")
	}
	if c.Objective != ObjectiveJudgeCapacity && c.Objective != ObjectiveAdmissionControl && c.Objective != ObjectiveMassiveBurst {
		return errors.New("--benchmark-objective must be judge-capacity, admission-control, or massive-burst")
	}
	if c.ObservationMode != ObservationFull && c.ObservationMode != ObservationAdmissionOnly && c.ObservationMode != ObservationRealistic {
		return errors.New("--observation-mode must be full, realistic, or admission-only")
	}
	if c.ResultRoot == "" || filepath.IsAbs(c.RunID) {
		return errors.New("result root/run ID is invalid")
	}
	return nil
}

func validLanguage(language string) bool {
	switch language {
	case "CPP", "GO", "PYTHON", "JAVA":
		return true
	default:
		return false
	}
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// PlannedSubmissions includes warmup. It never truncates a workload to fit a
// cap: Validate rejects a requested plan that exceeds the cap.
func (c Config) PlannedSubmissions() int {
	planned := c.WarmupCount
	maxInt := int(^uint(0) >> 1)
	switch c.Mode {
	case ModeBurst:
		if c.BurstSize > maxInt-planned {
			return maxInt
		}
		planned += c.BurstSize
	case ModeSustained:
		if c.TotalSubmissions > 0 {
			if c.TotalSubmissions > maxInt-planned {
				return maxInt
			}
			planned += c.TotalSubmissions
			return planned
		}
		if c.Rate == nil || c.Duration <= 0 {
			return planned
		}
		count := ceilRat(new(big.Rat).Mul(c.Rate, big.NewRat(int64(c.Duration), int64(time.Second))))
		if count > maxInt-planned {
			return maxInt
		}
		planned += count
	}
	return planned
}

func ceilRat(value *big.Rat) int {
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(value.Num(), value.Denom(), remainder)
	if remainder.Sign() != 0 && value.Sign() > 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if !quotient.IsInt64() || quotient.Int64() > int64(^uint(0)>>1) {
		return int(^uint(0) >> 1)
	}
	return int(quotient.Int64())
}

type enumValue struct {
	get func() string
	set func(string)
}

func (v enumValue) String() string {
	if v.get == nil {
		return ""
	}
	return v.get()
}
func (v enumValue) Set(value string) error {
	v.set(value)
	return nil
}

type urlValue struct{ cfg *Config }

func (v urlValue) String() string {
	if v.cfg == nil {
		return ""
	}
	return v.cfg.BaseURLRaw
}
func (v urlValue) Set(value string) error { return v.cfg.SetBaseURL(value) }
