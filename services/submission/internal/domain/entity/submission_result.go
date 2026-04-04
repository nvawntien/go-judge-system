package entity

import "time"

type ResultStatus string

const (
	ResultAccepted     ResultStatus = "ACCEPTED"
	ResultWrongAnswer  ResultStatus = "WRONG_ANSWER"
	ResultTimeLimit    ResultStatus = "TLE"
	ResultMemoryLimit  ResultStatus = "MLE"
	ResultRuntimeError ResultStatus = "RUNTIME_ERROR"
)

type SubmissionResult struct {
	ID            int64
	SubmissionID  int64
	TestIndex     int
	Status        ResultStatus
	ActualOutput  *string
	ExecutionTime *int
	MemoryUsed    *int
	CreatedAt     time.Time
}
