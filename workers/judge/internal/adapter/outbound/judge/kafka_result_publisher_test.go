package judge

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"go-judge-system/pkg/config"
	pkgjudge "go-judge-system/pkg/judge"
	"go-judge-system/workers/judge/internal/application/port/outbound"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type captureSyncProducer struct {
	message *sarama.ProducerMessage
}

func (p *captureSyncProducer) SendMessage(msg *sarama.ProducerMessage) (int32, int64, error) {
	p.message = msg
	return 1, 42, nil
}

func (p *captureSyncProducer) SendMessages(_ []*sarama.ProducerMessage) error { return nil }
func (p *captureSyncProducer) Close() error                                   { return nil }
func (p *captureSyncProducer) TxnStatus() sarama.ProducerTxnStatusFlag {
	return sarama.ProducerTxnFlagReady
}
func (p *captureSyncProducer) IsTransactional() bool { return false }
func (p *captureSyncProducer) BeginTxn() error       { return nil }
func (p *captureSyncProducer) CommitTxn() error      { return nil }
func (p *captureSyncProducer) AbortTxn() error       { return nil }
func (p *captureSyncProducer) AddOffsetsToTxn(_ map[string][]*sarama.PartitionOffsetMetadata, _ string) error {
	return nil
}
func (p *captureSyncProducer) AddMessageToTxn(_ *sarama.ConsumerMessage, _ string, _ *string) error {
	return nil
}

func TestPublishResultDoesNotIncludeHiddenTestcaseData(t *testing.T) {
	producer := &captureSyncProducer{}
	publisher := NewKafkaResultPublisher(producer, configWithResultTopic("judge.submission.results"), zap.NewNop())
	largeHidden := strings.Repeat("secret", 256*1024)

	err := publisher.PublishResult(context.Background(), 77, "attempt-77", &outbound.ExecutionResult{
		Status:        "WRONG_ANSWER",
		ExecutionTime: 12,
		MemoryUsed:    2048,
		TestCases: []outbound.TestCaseResult{{
			Index:          3,
			Status:         "WRONG_ANSWER",
			ActualOutput:   stringPtr(largeHidden),
			Input:          stringPtr("hidden input"),
			ExpectedOutput: stringPtr("hidden expected"),
			ExecutionTime:  12,
			MemoryUsed:     2048,
		}},
	})
	if err != nil {
		t.Fatalf("PublishResult() error = %v", err)
	}
	if producer.message == nil {
		t.Fatal("producer did not receive a message")
	}

	value, err := producer.message.Value.Encode()
	if err != nil {
		t.Fatalf("encode message: %v", err)
	}
	for _, secret := range []string{"secret", "hidden input", "hidden expected", "actual_output", "input", "expected_output"} {
		if strings.Contains(string(value), secret) {
			t.Fatalf("published payload leaked %q: %s", secret, string(value))
		}
	}

	var payload pkgjudge.ResultMessage
	if err := json.Unmarshal(value, &payload); err != nil {
		t.Fatalf("unmarshal result payload: %v", err)
	}
	if payload.SubmissionID != 77 || payload.AttemptID != "attempt-77" || payload.Status != "WRONG_ANSWER" {
		t.Fatalf("payload metadata = %+v", payload)
	}
	if len(payload.TestCases) != 1 {
		t.Fatalf("testcase count = %d, want 1", len(payload.TestCases))
	}
	testcase := payload.TestCases[0]
	if testcase.Index != 3 || testcase.Status != "WRONG_ANSWER" || testcase.ExecutionTime == nil || testcase.MemoryUsed == nil {
		t.Fatalf("testcase metadata = %+v", testcase)
	}
	if testcase.ActualOutput != nil || testcase.Input != nil || testcase.ExpectedOutput != nil {
		t.Fatalf("hidden testcase data must be omitted: %+v", testcase)
	}
}

func configWithResultTopic(topic string) config.KafkaConfig {
	return config.KafkaConfig{ResultTopic: topic}
}

func stringPtr(s string) *string {
	return &s
}
