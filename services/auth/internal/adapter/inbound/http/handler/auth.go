package handler

type AuthHandler struct {
	RegisterHandler *RegisterHandler
}

func NewAuthHandler(registerHandler *RegisterHandler) *AuthHandler {
	return &AuthHandler{
		RegisterHandler: registerHandler,
	}
}
