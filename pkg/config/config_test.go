package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfigDecodesJudgeTestcaseCache(t *testing.T) {
	dir := t.TempDir()
	contents := []byte(`testcase_cache:
  enabled: true
  max_bytes: 25165824
  max_entries: 256
  idle_ttl: 10m
  cleanup_interval: 30s
`)
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), contents, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	got := cfg.TestcaseCache
	if !got.Enabled || got.MaxBytes != 25_165_824 || got.MaxEntries != 256 || got.IdleTTL != 10*time.Minute || got.CleanupInterval != 30*time.Second {
		t.Fatalf("decoded testcase cache config = %#v", got)
	}
}

func TestLoadConfigMissingTestcaseCacheIsDisabled(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("server:\n  grpc_port: 9094\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.TestcaseCache.Enabled {
		t.Fatalf("missing testcase_cache decoded as enabled: %#v", cfg.TestcaseCache)
	}
}
