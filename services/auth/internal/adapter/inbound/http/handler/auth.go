package handler

import "go-judge-system/services/auth/internal/adapter/inbound/http/handler/auth"

type AuthHandler struct {
	Register *auth.RegisterHandler
}

func NewAuthHandler(
	registerHandler *auth.RegisterHandler,
) *AuthHandler {
	return &AuthHandler{
		Register: registerHandler,
	}
}
