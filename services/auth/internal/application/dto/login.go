package dto

type LoginRequest struct {
	Email    string `json:"email" binding:"omitempty,email"`
	Username string `json:"username" binding:"omitempty,min=3"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	AccessExpire int    `json:"access_expire"`
	RefreshToken string `json:"refresh_token"`
	RefreshExpire int    `json:"refresh_expire"`    
}
