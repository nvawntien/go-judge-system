// Package command implements the intentionally small judge-bench CLI.
package command

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/config"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/external"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/model"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/report"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/runner"
)

const usage = `Usage:
  judge-bench preflight --mode burst|sustained [flags]
  judge-bench burst [flags]
  judge-bench sustained [flags]
  judge-bench analyze --run-dir DIR [--container-stats FILE] [--kafka-lag FILE]
  judge-bench version

This tool is a controlled external client. It accepts only pre-issued cookie
sessions, never provisions users, and never retries a submission POST.`

func Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	switch args[0] {
	case "version":
		_, err := fmt.Fprintln(stdout, model.BenchmarkVersion)
		return err
	case "preflight":
		return preflight(ctx, args[1:], stdout, stderr)
	case "burst":
		return benchmark(ctx, config.ModeBurst, args[1:], stdout, stderr)
	case "sustained":
		return benchmark(ctx, config.ModeSustained, args[1:], stdout, stderr)
	case "analyze":
		return analyze(args[1:], stdout, stderr)
	case "help", "--help", "-h":
		_, err := fmt.Fprintln(stdout, usage)
		return err
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], usage)
	}
}

func preflight(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	mode := string(config.ModeBurst)
	fs, cfg := newFlagSet("preflight", config.ModeBurst, stderr)
	fs.StringVar(&mode, "mode", mode, "burst or sustained")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg.Mode = config.Mode(mode)
	if cfg.Mode != config.ModeBurst && cfg.Mode != config.ModeSustained {
		return errors.New("--mode must be burst or sustained")
	}
	if err := finalize(&cfg); err != nil {
		return err
	}
	prepared, err := runner.Preflight(ctx, cfg)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "preflight passed: mode=%s sessions=%d source_sha256=%s required_users=%d\n", cfg.Mode, len(prepared.Sessions), prepared.SourceSHA256, prepared.RequiredUsers)
	return err
}

func benchmark(ctx context.Context, mode config.Mode, args []string, stdout, stderr io.Writer) error {
	fs, cfg := newFlagSet(string(mode), mode, stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := finalize(&cfg); err != nil {
		return err
	}
	prepared, err := runner.Preflight(ctx, cfg)
	if err != nil {
		return err
	}
	run, err := runner.Run(ctx, prepared)
	if run != nil {
		_, _ = fmt.Fprintf(stdout, "benchmark result directory: %s\nclassification: %s\n", run.Dir, run.Summary.Classification)
	}
	return err
}

func newFlagSet(name string, mode config.Mode, stderr io.Writer) (*flag.FlagSet, config.Config) {
	cfg := config.Defaults(mode)
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	config.Bind(fs, &cfg)
	return fs, cfg
}

func finalize(cfg *config.Config) error {
	if err := cfg.ParseRate(cfg.RateRaw); err != nil {
		return err
	}
	if cfg.Seed == 0 {
		cfg.Seed = time.Now().UTC().UnixNano()
	}
	return cfg.Validate()
}

func analyze(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("analyze", flag.ContinueOnError)
	fs.SetOutput(stderr)
	runDir := fs.String("run-dir", "", "benchmark result directory")
	containerStats := fs.String("container-stats", "", "optional collector CSV")
	kafkaLag := fs.String("kafka-lag", "", "optional collector CSV")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *runDir == "" {
		return errors.New("--run-dir is required")
	}
	data, err := os.ReadFile(filepath.Join(*runDir, "summary.json"))
	if err != nil {
		return fmt.Errorf("read summary: %w", err)
	}
	var summary model.RunSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return fmt.Errorf("decode summary: %w", err)
	}
	runData, err := os.ReadFile(filepath.Join(*runDir, "run.json"))
	if err != nil {
		return fmt.Errorf("read run metadata: %w", err)
	}
	var metadata model.RunMetadata
	if err := json.Unmarshal(runData, &metadata); err != nil {
		return fmt.Errorf("decode run metadata: %w", err)
	}
	for _, path := range []string{*containerStats, *kafkaLag} {
		if path == "" {
			continue
		}
		samples, err := external.ReadCSV(path)
		if err != nil {
			return fmt.Errorf("read external metric %q: %w", path, err)
		}
		end := time.Time{}
		if metadata.EndedAt != nil {
			end = *metadata.EndedAt
		}
		fmt.Fprintf(stderr, "validated %d UTC external samples from %s; %d align with run wall-clock bounds\n", len(samples), path, external.CountInRange(samples, metadata.StartedAt, end))
	}
	_, err = fmt.Fprint(stdout, report.Markdown(summary))
	return err
}
