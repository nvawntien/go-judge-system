package dto

import "go-judge-system/pkg/rbac"

type UserIDRequest struct {
	UserID string `uri:"user_id" binding:"required,min=1"`
}

type AssignRoleRequest struct {
	Role rbac.Role `json:"role" binding:"required,oneof=user contributor moderator admin"`
}

type ListAdminUsersRequest struct {
	Page   *int   `form:"page"`
	Limit  *int   `form:"limit"`
	Search string `form:"search"`
	Role   string `form:"role"`
	Status string `form:"status"`
}

type AdminUserResponse struct {
	ID          string    `json:"id"`
	FullName    string    `json:"full_name"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	Role        rbac.Role `json:"role"`
	Rating      int       `json:"rating"`
	IsActive    bool      `json:"is_active"`
	IsSuspended bool      `json:"is_suspended"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	Bio         *string   `json:"bio,omitempty"`
	Country     *string   `json:"country,omitempty"`
	School      *string   `json:"school,omitempty"`
	Company     *string   `json:"company,omitempty"`
	GithubURL   *string   `json:"github_url,omitempty"`
	WebsiteURL  *string   `json:"website_url,omitempty"`
	LinkedinURL *string   `json:"linkedin_url,omitempty"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

type AdminUsersPaginationResponse struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type ListAdminUsersResponse struct {
	Items      []AdminUserResponse          `json:"items"`
	Pagination AdminUsersPaginationResponse `json:"pagination"`
}

type SetUserSuspensionRequest struct {
	Suspended *bool `json:"suspended" binding:"required"`
}
