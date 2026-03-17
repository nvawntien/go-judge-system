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
	TestCaseID    int64
	Status        ResultStatus
	ActualOutput  *string
	ExecutionTime *int
	MemoryUsed    *int
	Order         int
	CreatedAt     time.Time
}
