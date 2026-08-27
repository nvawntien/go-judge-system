package command

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/config"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/model"
)

func TestVersionDoesNotNeedTargetConfiguration(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Execute(context.Background(), []string{"version"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout.String()) != model.BenchmarkVersion {
		t.Fatalf("version output=%q", stdout.String())
	}
}

func TestSharedFlagsPopulateBoundConfig(t *testing.T) {
	var stderr bytes.Buffer
	fs, cfg := newFlagSet("sustained", config.ModeSustained, &stderr)
	if err := fs.Parse(sharedSustainedFlags("users.local.json")); err != nil {
		t.Fatal(err)
	}
	if err := finalize(cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL == nil || cfg.BaseURL.String() != "https://astracode.app" ||
		cfg.UsersFile != "users.local.json" || cfg.ProblemID != 3 ||
		cfg.ProblemSlug != "missing-number" || cfg.Language != "CPP" ||
		cfg.SourceFile != "/tmp/astracode-missing-number.cpp" ||
		cfg.ExpectedVerdict != "ACCEPTED" || cfg.SubmitCooldown != 3*time.Second ||
		!cfg.AllowRemote || cfg.ConfirmTargetHost != "astracode.app" ||
		cfg.MaxSubmissions != 10 || cfg.RateRaw != "0.10" || cfg.Rate == nil ||
		cfg.Rate.RatString() != "1/10" || cfg.Duration != time.Minute {
		t.Fatalf("shared flags did not populate the finalized config: %+v", *cfg)
	}
}

func TestSustainedExactVolumeFlagPopulatesBoundConfig(t *testing.T) {
	var stderr bytes.Buffer
	fs, cfg := newFlagSet("sustained", config.ModeSustained, &stderr)
	args := append([]string{}, sharedSustainedFlags("users.local.json")...)
	for index, argument := range args {
		if argument == "--duration" {
			args = append(args[:index], args[index+2:]...)
			break
		}
	}
	args = append(args, "--total-submissions", "10")
	if err := fs.Parse(args); err != nil {
		t.Fatal(err)
	}
	if err := finalize(cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.TotalSubmissions != 10 || cfg.Duration != 0 || cfg.Rate == nil {
		t.Fatalf("exact-volume flags did not populate the finalized config: %+v", *cfg)
	}
}

// These command-path tests deliberately use a missing credential file. A
// successful flag parse reaches that local validation before any HTTP call, so
// they prove the parsed Config is the one used by preflight/benchmark without
// contacting a target.
func TestCommandPathsUseParsedSharedFlagsBeforeNetwork(t *testing.T) {
	missingUsersFile := filepath.Join(t.TempDir(), "users.local.json")
	for _, test := range []struct {
		name string
		run  func(context.Context, []string, io.Writer, io.Writer) error
		args []string
	}{
		{
			name: "preflight sustained",
			run:  preflight,
			args: append([]string{"--mode", "sustained"}, sharedSustainedFlags(missingUsersFile)...),
		},
		{
			name: "sustained",
			run: func(ctx context.Context, args []string, stdout, stderr io.Writer) error {
				return benchmark(ctx, config.ModeSustained, args, stdout, stderr)
			},
			args: sharedSustainedFlags(missingUsersFile),
		},
		{
			name: "burst",
			run: func(ctx context.Context, args []string, stdout, stderr io.Writer) error {
				return benchmark(ctx, config.ModeBurst, args, stdout, stderr)
			},
			args: append(sharedSustainedFlags(missingUsersFile), "--burst-size", "1"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := test.run(t.Context(), test.args, &stdout, &stderr)
			if err == nil || !strings.Contains(err.Error(), "inspect credential file") {
				t.Fatalf("expected local credential-file validation after shared flag parsing, got %v", err)
			}
			if strings.Contains(err.Error(), "--base-url is required") {
				t.Fatalf("regression: parsed --base-url was lost: %v", err)
			}
		})
	}
}

func TestBenchmarkSubcommandHelpUsesFlagErrHelpWithoutCustomValuePanic(t *testing.T) {
	for _, test := range []struct {
		name  string
		args  []string
		flags []string
	}{
		{
			name:  "burst",
			args:  []string{"burst", "--help"},
			flags: []string{"base-url", "benchmark-objective", "session-refresh", "submit-error-policy", "system-config", "users-file", "max-submissions", "burst-size"},
		},
		{
			name:  "sustained",
			args:  []string{"sustained", "--help"},
			flags: []string{"base-url", "benchmark-objective", "session-refresh", "submit-error-policy", "system-config", "users-file", "max-submissions", "rate", "duration", "total-submissions"},
		},
		{
			name:  "preflight",
			args:  []string{"preflight", "--help"},
			flags: []string{"mode", "base-url", "benchmark-objective", "session-refresh", "submit-error-policy", "system-config", "users-file", "max-submissions"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := Execute(context.Background(), test.args, &stdout, &stderr)
			if !errors.Is(err, flag.ErrHelp) {
				t.Fatalf("help error=%v, want flag.ErrHelp; stderr=%q", err, stderr.String())
			}
			output := stderr.String()
			if strings.Contains(output, "panic calling String method") {
				t.Fatalf("help rendered a custom flag panic: %q", output)
			}
			for _, name := range test.flags {
				if !strings.Contains(output, "-"+name) {
					t.Fatalf("help output missing -%s: %q", name, output)
				}
			}
		})
	}
}

func sharedSustainedFlags(usersFile string) []string {
	return []string{
		"--base-url", "https://astracode.app",
		"--allow-remote",
		"--confirm-target-host", "astracode.app",
		"--users-file", usersFile,
		"--problem-id", "3",
		"--problem-slug", "missing-number",
		"--language", "CPP",
		"--source-file", "/tmp/astracode-missing-number.cpp",
		"--expected-verdict", "ACCEPTED",
		"--submit-cooldown", "3s",
		"--rate", "0.10",
		"--duration", "1m",
		"--max-submissions", "10",
	}
}
