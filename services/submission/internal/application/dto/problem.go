package dto

import "go-judge-system/pkg/rbac"

type ProblemActor struct {
	UserID string
	Role   rbac.Role
}

type ProblemMetadata struct {
	ID    int64
	Title string
	Slug  string
}
