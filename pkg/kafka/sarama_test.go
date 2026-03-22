package kafka

import (
	"reflect"
	"testing"

	"go-judge-system/pkg/config"

	"go.uber.org/zap"
)

func TestParseBrokers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{
			name: "empty string",
			raw:  "",
			want: nil,
		},
		{
			name: "spaces only",
			raw:  "   ",
			want: nil,
		},
		{
			name: "single broker",
			raw:  "kafka:9092",
			want: []string{"kafka:9092"},
		},
		{
			name: "multiple brokers with spaces and empties",
			raw:  " kafka-1:9092, ,kafka-2:9092 , kafka-3:9092 ",
			want: []string{"kafka-1:9092", "kafka-2:9092", "kafka-3:9092"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseBrokers(tt.raw)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseBrokers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewSyncProducer_EmptyBrokers(t *testing.T) {
	t.Parallel()

	_, err := NewSyncProducer(config.KafkaConfig{Brokers: ""}, zap.NewNop())
	if err == nil {
		t.Fatal("expected error for empty brokers, got nil")
	}
	if err.Error() != "kafka brokers are required" {
		t.Fatalf("unexpected error: %v", err)
	}
}
