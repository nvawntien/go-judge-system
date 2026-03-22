package judge

import "time"

// JobMessage represents a judge job published to Kafka.
// AttemptID is a UUID used as an idempotency key to prevent duplicate processing
// when a message is retried (e.g., due to network glitch or consumer rebalance).
type JobMessage struct {
	SubmissionID int64     `json:"submission_id"`
	ProblemID    int64     `json:"problem_id"`
	ProblemSlug  string    `json:"problem_slug,omitempty"`
	UserID       string    `json:"user_id"`
	Language     string    `json:"language"`
	SourceCode   string    `json:"source_code"`
	AttemptID    string    `json:"attempt_id"`
	EnqueuedAt   time.Time `json:"enqueued_at"`
}
