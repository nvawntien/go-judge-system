package entity

import (
	"strings"
	"time"
)

type AttemptTriggerType string

const (
	AttemptTriggerSubmission   AttemptTriggerType = "SUBMISSION"
	AttemptTriggerAdminRejudge AttemptTriggerType = "ADMIN_REJUDGE"
)

type SubmissionAttempt struct {
	ID                int64
	SubmissionID      int64
	AttemptID         string
	TriggerType       AttemptTriggerType
	TriggeredByUserID *string
	Status            Status
	TestcaseVersion   *int
	TestCount         *int
	DatasetChecksum   *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func NewSubmissionAttempt(
	submissionID int64,
	attemptID string,
	triggerType AttemptTriggerType,
	triggeredByUserID string,
) *SubmissionAttempt {
	now := time.Now()
	return &SubmissionAttempt{
		SubmissionID:      submissionID,
		AttemptID:         strings.TrimSpace(attemptID),
		TriggerType:       triggerType,
		TriggeredByUserID: nonBlankStringPointer(triggeredByUserID),
		Status:            StatusPending,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func nonBlankStringPointer(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func (a *SubmissionAttempt) MarkCompleted(status Status, testcaseVersion *int, testCount *int, datasetChecksum *string) {
	a.Status = status
	a.TestcaseVersion = testcaseVersion
	a.TestCount = testCount
	a.DatasetChecksum = datasetChecksum
	a.UpdatedAt = time.Now()
}
