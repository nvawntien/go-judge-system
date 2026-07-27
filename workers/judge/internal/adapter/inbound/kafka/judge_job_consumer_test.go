package kafka

import (
	"context"
	"encoding/json"
	"testing"

	pkgjudge "go-judge-system/pkg/judge"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type fakeProcessJudgeJobUseCase struct {
	got *pkgjudge.JobMessage
}

func (f *fakeProcessJudgeJobUseCase) Execute(_ context.Context, job *pkgjudge.JobMessage) error {
	cp := *job
	f.got = &cp
	return nil
}

type fakeConsumerGroupSession struct {
	ctx      context.Context
	marked   bool
	metadata string
}

func (f *fakeConsumerGroupSession) Claims() map[string][]int32 { return nil }
func (f *fakeConsumerGroupSession) MemberID() string           { return "member-1" }
func (f *fakeConsumerGroupSession) GenerationID() int32        { return 1 }
func (f *fakeConsumerGroupSession) MarkOffset(string, int32, int64, string) {
}
func (f *fakeConsumerGroupSession) Commit() {}
func (f *fakeConsumerGroupSession) ResetOffset(string, int32, int64, string) {
}
func (f *fakeConsumerGroupSession) MarkMessage(_ *sarama.ConsumerMessage, metadata string) {
	f.marked = true
	f.metadata = metadata
}
func (f *fakeConsumerGroupSession) Context() context.Context {
	if f.ctx == nil {
		return context.Background()
	}
	return f.ctx
}

func TestJudgeJobHandlerDecodesCallsUseCaseAndMarksMessage(t *testing.T) {
	t.Parallel()

	payload := pkgjudge.JobMessage{
		SubmissionID: 99,
		ProblemID:    42,
		AttemptID:    "attempt-a",
		Language:     "GO",
		SourceCode:   "package main\nfunc main(){}",
	}
	value, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	useCase := &fakeProcessJudgeJobUseCase{}
	session := &fakeConsumerGroupSession{}
	handler := &judgeJobHandler{
		useCase:    useCase,
		maxRetries: 1,
		logger:     zap.NewNop(),
	}
	handler.handleMessage(session, &sarama.ConsumerMessage{Value: value})

	if useCase.got == nil || useCase.got.SubmissionID != 99 || useCase.got.AttemptID != "attempt-a" {
		t.Fatalf("use case payload = %#v", useCase.got)
	}
	if !session.marked || session.metadata != "processed" {
		t.Fatalf("mark = %v/%q, want processed", session.marked, session.metadata)
	}
}
