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
)

type Config struct {
	Mode                    Mode
	BaseURL                 *url.URL
	BaseURLRaw              string
	UsersFile               string
	ProblemID               int64
	ProblemSlug             string
	Language                string
	SourceFile              string
	ExpectedVerdict         string
	ResultRoot              string
	RunID                   string
	Repetition              int
	Seed                    int64
	WarmupCount             int
	WarmupTimeout           time.Duration
	SubmitCooldown          time.Duration
	CooldownGuard           time.Duration
	SubmitLatencyBudget     time.Duration
	PoolHeadroomPercent     float64
	APITimeout              time.Duration
	SubmissionTimeout       time.Duration
	DrainTimeout            time.Duration
	Window                  time.Duration
	MaxSubmissions          int
	MaxInFlight             int
	ErrorPolicy             ErrorPolicy
	Objective               Objective
	SSEConnectTimeout       time.Duration
	SSEIdleTimeout          time.Duration
	SSEMaxReconnects        int
	SSEBackoffBase          time.Duration
	SSEBackoffMax           time.Duration
	SafetyReconcileInterval time.Duration
	ReconcileMaxQPS         float64
	RefreshMode             RefreshMode
	AuthValidityMargin      time.Duration
	AllowRemote             bool
	ConfirmTargetHost       string
	BurstSize               int
	Jitter                  time.Duration
	ScheduleDelayP95Budget  time.Duration
	Rate                    *big.Rat
	RateRaw                 string
	Duration                time.Duration
}

func Defaults(mode Mode) Config {
	return Config{
		Mode:                    mode,
		ResultRoot:              "bench-results",
		Repetition:              1,
		WarmupTimeout:           2 * time.Minute,
		CooldownGuard:           100 * time.Millisecond,
		SubmitLatencyBudget:     500 * time.Millisecond,
		PoolHeadroomPercent:     25,
		APITimeout:              5 * time.Second,
		SubmissionTimeout:       5 * time.Minute,
		DrainTimeout:            5 * time.Minute,
		Window:                  30 * time.Second,
		MaxInFlight:             100,
		ErrorPolicy:             ErrorPolicyStop,
		Objective:               ObjectiveJudgeCapacity,
		SSEConnectTimeout:       10 * time.Second,
		SSEIdleTimeout:          45 * time.Second,
		SSEMaxReconnects:        5,
		SSEBackoffBase:          time.Second,
		SSEBackoffMax:           10 * time.Second,
		SafetyReconcileInterval: 60 * time.Second,
		ReconcileMaxQPS:         2,
		RefreshMode:             RefreshAuto,
		AuthValidityMargin:      time.Minute,
		ScheduleDelayP95Budget:  25 * time.Millisecond,
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
	fs.IntVar(&cfg.MaxInFlight, "max-in-flight", cfg.MaxInFlight, "accepted unresolved submission cap")
	fs.Var(enumValue{get: func() string { return string(cfg.ErrorPolicy) }, set: func(value string) { cfg.ErrorPolicy = ErrorPolicy(value) }}, "submit-error-policy", "stop or continue")
	fs.Var(enumValue{get: func() string { return string(cfg.Objective) }, set: func(value string) { cfg.Objective = Objective(value) }}, "benchmark-objective", "judge-capacity or admission-control")
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
	if c.Mode == ModeBurst && c.BurstSize <= 0 {
		return errors.New("--burst-size must be positive for burst mode")
	}
	if c.Mode == ModeSustained && (c.Rate == nil || c.Duration <= 0) {
		return errors.New("sustained mode requires positive --rate and --duration")
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
	if c.WarmupCount < 0 || c.Repetition <= 0 || c.MaxInFlight <= 0 || c.PoolHeadroomPercent < 0 {
		return errors.New("counts and pool headroom are invalid")
	}
	if c.APITimeout <= 0 || c.SubmissionTimeout <= 0 || c.DrainTimeout <= 0 || c.Window <= 0 || c.WarmupTimeout <= 0 {
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
	if c.Objective != ObjectiveJudgeCapacity && c.Objective != ObjectiveAdmissionControl {
		return errors.New("--benchmark-objective must be judge-capacity or admission-control")
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
	switch c.Mode {
	case ModeBurst:
		planned += c.BurstSize
	case ModeSustained:
		if c.Rate == nil || c.Duration <= 0 {
			return planned
		}
		count := ceilRat(new(big.Rat).Mul(c.Rate, big.NewRat(int64(c.Duration), int64(time.Second))))
		maxInt := int(^uint(0) >> 1)
		if count > maxInt-planned {
			return maxInt
		}
		planned += count
	}
	return planned
}

// OffsetFor derives each arrival from its original origin. It deliberately
// avoids repeatedly adding a rounded duration.
func OffsetFor(rate *big.Rat, index int) time.Duration {
	if rate == nil || index < 0 {
		return 0
	}
	seconds := new(big.Rat).Quo(big.NewRat(int64(index), 1), rate)
	nanoseconds := new(big.Rat).Mul(seconds, big.NewRat(int64(time.Second), 1))
	value := new(big.Int).Quo(nanoseconds.Num(), nanoseconds.Denom())
	if !value.IsInt64() || value.Int64() > int64(^uint(0)>>1) {
		return time.Duration(int64(^uint(0) >> 1))
	}
	return time.Duration(value.Int64())
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

func (v enumValue) String() string { return v.get() }
func (v enumValue) Set(value string) error {
	v.set(value)
	return nil
}

type urlValue struct{ cfg *Config }

func (v urlValue) String() string         { return v.cfg.BaseURLRaw }
func (v urlValue) Set(value string) error { return v.cfg.SetBaseURL(value) }
