package dto

type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=8"`
	ResetToken  string `json:"reset_token" binding:"required"`
}