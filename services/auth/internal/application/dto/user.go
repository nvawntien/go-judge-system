package dto

import "go-judge-system/pkg/rbac"

type GetMeRequest struct {
}

type GetMeResponse struct {
	ID        string    `json:"id"`
	FullName  string    `json:"full_name"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      rbac.Role `json:"role"`
	Rating    int       `json:"rating"`
	IsActive  bool      `json:"is_active"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

type GetProfileRequest struct {
	Username string `uri:"username" binding:"required"`
}

type GetProfileResponse struct {
	FullName  string `json:"full_name"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Rating    int    `json:"rating"`
	CreatedAt string `json:"created_at"`
}
