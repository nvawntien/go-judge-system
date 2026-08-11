package dto

import (
	"go-judge-system/pkg/rbac"
	"mime/multipart"
)

type GetMeRequest struct {
}

type GetMeResponse struct {
	ID          string    `json:"id"`
	FullName    string    `json:"full_name"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	Role        rbac.Role `json:"role"`
	Rating      int       `json:"rating"`
	IsActive    bool      `json:"is_active"`
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

type GetProfileRequest struct {
	Username string `uri:"username" binding:"required"`
}

type GetProfileResponse struct {
	FullName    string  `json:"full_name"`
	Username    string  `json:"username"`
	Rating      int     `json:"rating"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	Bio         *string `json:"bio,omitempty"`
	Country     *string `json:"country,omitempty"`
	School      *string `json:"school,omitempty"`
	Company     *string `json:"company,omitempty"`
	GithubURL   *string `json:"github_url,omitempty"`
	WebsiteURL  *string `json:"website_url,omitempty"`
	LinkedinURL *string `json:"linkedin_url,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

type UpdateProfileRequest struct {
	FullName    *string `json:"full_name"`
	Bio         *string `json:"bio"`
	Country     *string `json:"country"`
	School      *string `json:"school"`
	Company     *string `json:"company"`
	GithubURL   *string `json:"github_url"`
	WebsiteURL  *string `json:"website_url"`
	LinkedinURL *string `json:"linkedin_url"`
}

type UploadAvatarRequest struct {
	Avatar *multipart.FileHeader `form:"avatar" binding:"required"`
}

type UploadAvatarResponse struct {
	AvatarURL string `json:"avatarUrl"`
}
