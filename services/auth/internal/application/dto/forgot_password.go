package dto

type ForgotPasswordRequest struct {
	Email   string `json:"email" binding:"required,email"`
}
