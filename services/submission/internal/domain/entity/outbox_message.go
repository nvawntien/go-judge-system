package entity

import "time"

const (
	OutboxStatusPending   = "PENDING"
	OutboxStatusPublished = "PUBLISHED"
	OutboxStatusFailed    = "FAILED"
)

type OutboxMessage struct {
	ID          int64
	AggregateID int64
	Topic       string
	Payload     []byte
	Status      string
	CreatedAt   time.Time
	PublishedAt *time.Time
	RetryCount  int
	ErrorReason *string
}
