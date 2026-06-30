package dto

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required,min=8"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required,min=8,eqfield=NewPassword"`
}

type ForgotPasswordRequest struct {
	Email   string `json:"email" binding:"required,email"`
}

type LoginRequest struct {
	Identifier string `json:"identifier" binding:"required"` // can be email or username
	Password   string `json:"password" binding:"required"`
}

type LoginResponse struct {
	AccessToken   string `json:"access_token"`
	AccessExpire  int    `json:"access_expire"`
	RefreshToken  string `json:"refresh_token"`
	RefreshExpire int    `json:"refresh_expire"`
}

type RegisterRequest struct {
	FullName string `json:"full_name" binding:"required"`
	Username string `json:"username" binding:"required,min=3"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token           string `json:"token" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required,min=8,eqfield=NewPassword"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

