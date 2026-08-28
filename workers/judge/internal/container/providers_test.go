package container

import (
	"strings"
	"testing"
	"time"

	"go-judge-system/pkg/config"
)

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
