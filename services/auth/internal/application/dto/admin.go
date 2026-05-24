package dto

import "go-judge-system/pkg/rbac"

type UserIDRequest struct {
	UserID string `uri:"user_id" binding:"required,min=1"`
}

type AssignRoleRequest struct {
	Role rbac.Role `json:"role" binding:"required,oneof=user contributor moderator admin"`
}