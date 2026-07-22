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
