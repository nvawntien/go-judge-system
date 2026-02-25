package dto

type VerifyOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required,len=6"`
	Purpose string `json:"purpose" binding:"required,oneof=activation forgot_password"`
}
