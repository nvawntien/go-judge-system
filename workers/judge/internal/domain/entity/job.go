package entity

import (
	"time"
)

type ExecutionStatus string

const (
	ExecutionStatusRunning       ExecutionStatus = "RUNNING"
	ExecutionStatusCompleted     ExecutionStatus = "COMPLETED"
	ExecutionStatusTimeoutError  ExecutionStatus = "TIMEOUT_ERROR"
	ExecutionStatusRuntimeError  ExecutionStatus = "RUNTIME_ERROR"
	ExecutionStatusInternalError ExecutionStatus = "INTERNAL_ERROR"
)

// Job represents a judging task received from submission service
type Job struct {
	SubmissionID  int64
	ProblemID     int64
	UserID        string
	Language      string
	SourceCode    string
	Status        ExecutionStatus
	StartedAt     *time.Time
	CompletedAt   *time.Time
	TimeoutAt     time.Time
	Error         *string
}

func NewJob(submissionID, problemID int64, userID, language, sourceCode string, timeout time.Duration) *Job {
	now := time.Now()
	return &Job{
		SubmissionID: submissionID,
		ProblemID:    problemID,
		UserID:       userID,
		Language:     language,
		SourceCode:   sourceCode,
		Status:       ExecutionStatusRunning,
		TimeoutAt:    now.Add(timeout),
	}
}

func (j *Job) MarkStarted() {
	now := time.Now()
	j.StartedAt = &now
	j.Status = ExecutionStatusRunning
}

func (j *Job) MarkCompleted() {
	now := time.Now()
	j.CompletedAt = &now
	j.Status = ExecutionStatusCompleted
}

func (j *Job) MarkError(errMsg string) {
	now := time.Now()
	j.CompletedAt = &now
	j.Error = &errMsg
	j.Status = ExecutionStatusInternalError
}

func (j *Job) IsTimedOut() bool {
	return time.Now().After(j.TimeoutAt)
}
