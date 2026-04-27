package dto

type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}
