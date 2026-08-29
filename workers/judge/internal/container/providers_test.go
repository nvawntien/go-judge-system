package container

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go-judge-system/pkg/config"
)

func TestProvideTestcaseCacheConfigFromLoadedConfig(t *testing.T) {
	for _, tt := range []struct{ name, yaml, want string }{
		{"valid", "testcase_cache:\n  enabled: true\n  max_bytes: 25165824\n  max_entries: 256\n  idle_ttl: 10m\n  cleanup_interval: 30s\n", ""},
		{"absent", "server:\n  grpc_port: 9094\n", ""},
		{"zero bytes", "testcase_cache:\n  enabled: true\n  max_bytes: 0\n  max_entries: 1\n  cleanup_interval: 1s\n", "max_bytes"},
		{"zero entries", "testcase_cache:\n  enabled: true\n  max_bytes: 1\n  max_entries: 0\n  cleanup_interval: 1s\n", "max_entries"},
		{"zero interval", "testcase_cache:\n  enabled: true\n  max_bytes: 1\n  max_entries: 1\n  cleanup_interval: 0s\n", "cleanup_interval"},
		{"negative idle", "testcase_cache:\n  enabled: true\n  max_bytes: 1\n  max_entries: 1\n  idle_ttl: -1s\n  cleanup_interval: 1s\n", "idle_ttl"},
		{"disabled", "testcase_cache:\n  enabled: false\n  max_bytes: 0\n  max_entries: 0\n", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(tt.yaml), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := config.LoadConfig(dir)
			if err != nil {
				t.Fatal(err)
			}
			got, err := ProvideTestcaseCacheConfig(cfg)
			if tt.want != "" {
				if err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("error=%v want=%s", err, tt.want)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.name == "valid" && (!got.Enabled || got.MaxBytes != 25165824 || got.MaxEntries != 256 || got.IdleTTL != 10*time.Minute || got.CleanupInterval != 30*time.Second) {
				t.Fatalf("got=%#v", got)
			}
		})
	}
	// mapstructure rejects malformed duration strings during the shared Viper load.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("testcase_cache:\n  enabled: true\n  idle_ttl: nonsense\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := config.LoadConfig(dir); err == nil {
		t.Fatal("LoadConfig accepted malformed duration")
	}
}

func TestProvideTestcaseCacheConfig(t *testing.T) {
	valid := config.TestcaseCacheConfig{
		Enabled: true, MaxBytes: 1024, MaxEntries: 8, IdleTTL: time.Minute, CleanupInterval: time.Second,
	}
	if got, err := ProvideTestcaseCacheConfig(&config.Config{TestcaseCache: valid}); err != nil || got != valid {
		t.Fatalf("valid testcase cache config = %#v/%v", got, err)
	}

	productionInitial := config.TestcaseCacheConfig{
		Enabled: true, MaxBytes: 25_165_824, MaxEntries: 256, IdleTTL: 10 * time.Minute, CleanupInterval: 30 * time.Second,
	}
	if got, err := ProvideTestcaseCacheConfig(&config.Config{TestcaseCache: productionInitial}); err != nil || got != productionInitial {
		t.Fatalf("initial production testcase cache config = %#v/%v", got, err)
	}

	for _, tt := range []struct {
		name string
		cfg  config.TestcaseCacheConfig
		want string
	}{
		{"max bytes", config.TestcaseCacheConfig{Enabled: true, MaxEntries: 1, CleanupInterval: time.Second}, "max_bytes"},
		{"max entries", config.TestcaseCacheConfig{Enabled: true, MaxBytes: 1, CleanupInterval: time.Second}, "max_entries"},
		{"negative idle ttl", config.TestcaseCacheConfig{Enabled: true, MaxBytes: 1, MaxEntries: 1, IdleTTL: -time.Second, CleanupInterval: time.Second}, "idle_ttl"},
		{"cleanup interval", config.TestcaseCacheConfig{Enabled: true, MaxBytes: 1, MaxEntries: 1}, "cleanup_interval"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ProvideTestcaseCacheConfig(&config.Config{TestcaseCache: tt.cfg})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want field %q", err, tt.want)
			}
		})
	}

	if got, err := ProvideTestcaseCacheConfig(&config.Config{}); err != nil || got.Enabled {
		t.Fatalf("disabled zero config = %#v/%v", got, err)
	}
}
