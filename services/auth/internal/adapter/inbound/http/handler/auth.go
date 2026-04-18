package handler

import "go-judge-system/services/auth/internal/adapter/inbound/http/handler/auth"

type AuthHandler struct {
	Register    *auth.RegisterHandler
	VerifyEmail *auth.VerifyEmailHandler
}

func NewAuthHandler(
	registerHandler *auth.RegisterHandler,
	verifyEmailHandler *auth.VerifyEmailHandler,
) *AuthHandler {
	return &AuthHandler{
		Register:    registerHandler,
		VerifyEmail: verifyEmailHandler,
	}
}
