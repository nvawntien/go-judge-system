package dto

type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}
