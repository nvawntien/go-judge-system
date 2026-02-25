package dto

type ResendOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	Purpose string `json:"purpose" binding:"required,oneof=activation forgot_password"`
}