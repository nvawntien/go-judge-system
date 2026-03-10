package entity

import "time"

type TestCase struct {
	ID             int64
	ProblemID      int64
	Input          string
	ExpectedOutput string
	IsExample      bool
	Order          int
	CreatedAt      time.Time
}

func NewTestCase(problemID int64, input, expectedOutput string, isExample bool, order int) *TestCase {
	return &TestCase{
		ProblemID:      problemID,
		Input:          input,
		ExpectedOutput: expectedOutput,
		IsExample:      isExample,
		Order:          order,
		CreatedAt:      time.Now(),
	}
}
