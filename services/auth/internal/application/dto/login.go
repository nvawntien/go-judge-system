package dto

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
