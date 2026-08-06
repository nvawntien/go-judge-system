package dto

import "go-judge-system/pkg/rbac"

type ProblemActor struct {
	UserID string
	Role   rbac.Role
}

type ProblemMetadata struct {
	ID          int64
	Title       string
	Slug        string
	TimeLimit   float64
	MemoryLimit int
}

type ProblemTestCaseMetadata struct {
	ProblemID int64
	TestCount int
	Version   int
}
