package config

import (
	"math/big"
	"net/url"
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

func TestOffsetForHasNoAccumulatedDrift(t *testing.T) {
	rate, _ := new(big.Rat).SetString("0.3")
	if got := OffsetFor(rate, 2); got != 6666666666*time.Nanosecond {
		t.Fatalf("offset=%s", got)
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
