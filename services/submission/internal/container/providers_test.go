package container

import (
	"testing"
	"time"

	"go-judge-system/pkg/config"
)

func TestProvideProblemGRPCConfigDefaults(t *testing.T) {
	got, err := ProvideProblemGRPCConfig(&config.Config{})
	if err != nil {
		t.Fatalf("ProvideProblemGRPCConfig() error = %v", err)
	}
	if got.Address != "problem-service:9092" || got.Timeout != time.Second {
		t.Fatalf("config = %+v", got)
	}
}

func TestProvideProblemGRPCConfigPreservesValidOverrides(t *testing.T) {
	want := config.ProblemGRPCConfig{Address: "localhost:19092", Timeout: 250 * time.Millisecond}
	got, err := ProvideProblemGRPCConfig(&config.Config{ProblemGRPC: want})
	if err != nil {
		t.Fatalf("ProvideProblemGRPCConfig() error = %v", err)
	}
	if got != want {
		t.Fatalf("config = %+v, want %+v", got, want)
	}
}

func TestProvideProblemGRPCConfigRejectsNegativeTimeout(t *testing.T) {
	_, err := ProvideProblemGRPCConfig(&config.Config{ProblemGRPC: config.ProblemGRPCConfig{
		Address: "problem-service:9092",
		Timeout: -time.Second,
	}})
	if err == nil {
		t.Fatal("ProvideProblemGRPCConfig() error = nil, want validation error")
	}
}

func TestProvideSSEConfigValidatesAndDefaults(t *testing.T) {
	got, err := ProvideSSEConfig(&config.Config{SSE: config.SSEConfig{
		TicketSecret:      "  local-secret  ",
		TicketTTL:         2 * time.Minute,
		HeartbeatInterval: 15 * time.Second,
	}})
	if err != nil {
		t.Fatalf("ProvideSSEConfig() error = %v", err)
	}
	if got.TicketSecret != "local-secret" ||
		got.TicketTTL != 2*time.Minute ||
		got.HeartbeatInterval != 15*time.Second ||
		got.AllowedOrigin != "http://localhost:3000" {
		t.Fatalf("config = %+v", got)
	}
}

func TestProvideSSEConfigRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.SSEConfig
	}{
		{
			name: "empty secret",
			cfg: config.SSEConfig{
				TicketTTL:         2 * time.Minute,
				HeartbeatInterval: 15 * time.Second,
			},
		},
		{
			name: "zero ttl",
			cfg: config.SSEConfig{
				TicketSecret:      "secret",
				HeartbeatInterval: 15 * time.Second,
			},
		},
		{
			name: "zero heartbeat",
			cfg: config.SSEConfig{
				TicketSecret: "secret",
				TicketTTL:    2 * time.Minute,
			},
		},
		{
			name: "heartbeat exceeds edge proxy timeout",
			cfg: config.SSEConfig{
				TicketSecret:      "secret",
				TicketTTL:         2 * time.Minute,
				HeartbeatInterval: 75 * time.Second,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ProvideSSEConfig(&config.Config{SSE: tt.cfg})
			if err == nil {
				t.Fatal("ProvideSSEConfig() error = nil, want validation error")
			}
		})
	}
}
