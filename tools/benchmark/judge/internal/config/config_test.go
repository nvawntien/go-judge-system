package config

import (
	"math/big"
	"net/url"
	"strings"
	"testing"
	"time"
)

func valid(t *testing.T, mode Mode) Config {
	t.Helper()
	cfg := Defaults(mode)
	u, err := url.Parse("http://127.0.0.1:8080")
	if err != nil {
		t.Fatal(err)
	}
	cfg.BaseURL = u
	cfg.BaseURLRaw = u.String()
	cfg.UsersFile = "users.json"
	cfg.ProblemID = 1
	cfg.ProblemSlug = "two-sum"
	cfg.Language = "GO"
	cfg.SourceFile = "main.go"
	cfg.ExpectedVerdict = "ACCEPTED"
	cfg.SubmitCooldown = time.Second
	cfg.MaxSubmissions = 100
	if mode == ModeBurst {
		cfg.BurstSize = 1
	} else {
		cfg.Rate = big.NewRat(3, 10)
		cfg.RateRaw = "0.3"
		cfg.Duration = time.Minute
	}
	return cfg
}

func TestValidateRemoteSafetyAndSubmissionCap(t *testing.T) {
	cfg := valid(t, ModeBurst)
	u, _ := url.Parse("https://benchmark.example.test")
	cfg.BaseURL = u
	cfg.BaseURLRaw = u.String()
	if err := cfg.Validate(); err == nil {
		t.Fatal("unconfirmed remote target accepted")
	}
	cfg.AllowRemote = true
	cfg.ConfirmTargetHost = "benchmark.example.test"
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	cfg.MaxSubmissions = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("remote target without cap accepted")
	}
}

func TestValidateRejectsImpossibleReconciliationCapacity(t *testing.T) {
	cfg := valid(t, ModeBurst)
	cfg.MaxInFlight = 100
	cfg.SafetyReconcileInterval = time.Second
	cfg.ReconcileMaxQPS = 2
	if err := cfg.Validate(); err == nil {
		t.Fatal("impossible reconciliation backlog accepted")
	}
}

func TestValidateRejectsInvalidCooldownAndSubmissionCap(t *testing.T) {
	cfg := valid(t, ModeBurst)
	cfg.SubmitCooldown = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("zero cooldown accepted")
	}
	cfg = valid(t, ModeBurst)
	cfg.MaxSubmissions = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("zero submission cap accepted")
	}
	if err := cfg.ParseRate("not-a-rate"); err == nil {
		t.Fatal("invalid rate accepted")
	}
}

func TestValidateExactVolumeSustainedSafetyBudget(t *testing.T) {
	for _, test := range []struct {
		name    string
		warmup  int
		total   int
		max     int
		wantErr bool
	}{
		{name: "exact cap boundary", total: 1000, max: 1000},
		{name: "warmup included in cap", warmup: 5, total: 1000, max: 1005},
		{name: "warmup exceeds cap", warmup: 5, total: 1000, max: 1000, wantErr: true},
		{name: "total exceeds cap", total: 1001, max: 1000, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := valid(t, ModeSustained)
			cfg.Duration = 0
			cfg.WarmupCount, cfg.TotalSubmissions, cfg.MaxSubmissions = test.warmup, test.total, test.max
			err := cfg.Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("Validate() error=%v wantErr=%v", err, test.wantErr)
			}
			if !test.wantErr && cfg.PlannedSubmissions() != test.warmup+test.total {
				t.Fatalf("planned=%d", cfg.PlannedSubmissions())
			}
		})
	}
}

func TestValidateExactVolumeSchedulingForms(t *testing.T) {
	base := func() Config {
		cfg := valid(t, ModeSustained)
		cfg.Duration = 0
		cfg.TotalSubmissions = 2
		cfg.MaxSubmissions = 2
		return cfg
	}
	for _, test := range []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{name: "duration and total conflict", mutate: func(c *Config) { c.Duration = time.Second }, wantErr: "either --duration or --total-submissions"},
		{name: "neither duration nor total", mutate: func(c *Config) { c.TotalSubmissions = 0 }, wantErr: "--duration or --total-submissions"},
		{name: "negative total", mutate: func(c *Config) { c.TotalSubmissions = -1 }, wantErr: "must not be negative"},
		{name: "rate required", mutate: func(c *Config) { c.Rate = nil }, wantErr: "requires positive --rate"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := base()
			test.mutate(&cfg)
			if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Validate() error=%v, want %q", err, test.wantErr)
			}
		})
	}
	cfg := valid(t, ModeBurst)
	cfg.TotalSubmissions = 1
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "only valid for sustained") {
		t.Fatalf("burst accepted total submissions: %v", err)
	}
}

func TestValidateMassiveBurstRequiresDistinctSelectedUsersAndInFlightCapacity(t *testing.T) {
	cfg := valid(t, ModeBurst)
	cfg.Objective = ObjectiveMassiveBurst
	cfg.BurstSize = 1000
	cfg.UserCount = 1000
	cfg.MaxInFlight = 1000
	cfg.MaxSubmissions = 1000
	cfg.SafetyReconcileInterval = 0
	cfg.ErrorPolicy = ErrorPolicyContinue
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Config){
		func(c *Config) { c.UserCount = 999 },
		func(c *Config) { c.UserCount = 100001 },
		func(c *Config) { c.MaxInFlight = 999 },
		func(c *Config) { c.WarmupCount = 1; c.MaxSubmissions = 1001 },
		func(c *Config) { c.Jitter = time.Millisecond },
		func(c *Config) { c.ErrorPolicy = ErrorPolicyStop },
	} {
		copy := cfg
		mutate(&copy)
		if err := copy.Validate(); err == nil {
			t.Fatalf("unsafe massive burst accepted: %+v", copy)
		}
	}
}

func TestAdmissionOnlyRequiresMassiveBurstAndSupports100K(t *testing.T) {
	cfg := valid(t, ModeBurst)
	cfg.Objective = ObjectiveMassiveBurst
	cfg.ObservationMode = ObservationAdmissionOnly
	cfg.BurstSize, cfg.UserCount, cfg.MaxInFlight, cfg.MaxSubmissions = 100000, 100000, 100000, 100000
	cfg.WarmupCount, cfg.Jitter, cfg.SafetyReconcileInterval, cfg.ErrorPolicy = 0, 0, 0, ErrorPolicyContinue
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	cfg.AdmissionPreflightSample = 100001
	if err := cfg.Validate(); err == nil {
		t.Fatal("oversized sampled preflight accepted")
	}
	cfg = valid(t, ModeBurst)
	cfg.ObservationMode = ObservationAdmissionOnly
	if err := cfg.Validate(); err == nil {
		t.Fatal("admission-only outside massive burst accepted")
	}
}

func TestRealisticObservationRequiresMassiveBurstAndPositiveHold(t *testing.T) {
	cfg := valid(t, ModeBurst)
	cfg.Objective, cfg.ObservationMode = ObjectiveMassiveBurst, ObservationRealistic
	cfg.BurstSize, cfg.UserCount, cfg.MaxInFlight, cfg.MaxSubmissions = 100000, 100000, 100000, 100000
	cfg.WarmupCount, cfg.Jitter, cfg.SafetyReconcileInterval, cfg.ErrorPolicy = 0, 0, 0, ErrorPolicyContinue
	cfg.SSEHoldDuration = 30 * time.Second
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	cfg.SSEHoldDuration = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("realistic mode accepted zero hold duration")
	}
	cfg = valid(t, ModeBurst)
	cfg.ObservationMode = ObservationRealistic
	if err := cfg.Validate(); err == nil {
		t.Fatal("realistic mode outside massive burst accepted")
	}
}

func TestBurstPlanningOverflowCannotBypassSubmissionCap(t *testing.T) {
	cfg := valid(t, ModeBurst)
	cfg.BurstSize = int(^uint(0) >> 1)
	cfg.WarmupCount = 1
	if cfg.PlannedSubmissions() != int(^uint(0)>>1) {
		t.Fatalf("planned overflowed: %d", cfg.PlannedSubmissions())
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("overflowing burst plan bypassed max submission cap")
	}
}

func TestCustomFlagValuesHaveSafeZeroValueString(t *testing.T) {
	if got := (urlValue{}).String(); got != "" {
		t.Fatalf("zero urlValue String()=%q, want empty", got)
	}
	if got := (enumValue{}).String(); got != "" {
		t.Fatalf("zero enumValue String()=%q, want empty", got)
	}
}
