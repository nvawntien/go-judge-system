package handler

import "go-judge-system/services/auth/internal/adapter/inbound/http/handler/auth"

type AuthHandler struct {
	Register    *auth.RegisterHandler
	VerifyEmail *auth.VerifyEmailHandler
	Login       *auth.LoginHandler
	Logout      *auth.LogoutHandler
}

func NewAuthHandler(
	registerHandler *auth.RegisterHandler,
	verifyEmailHandler *auth.VerifyEmailHandler,
	loginHandler *auth.LoginHandler,
	logoutHandler *auth.LogoutHandler,
) *AuthHandler {
	return &AuthHandler{
		Register:    registerHandler,
		VerifyEmail: verifyEmailHandler,
		Login:       loginHandler,
		Logout:      logoutHandler,
	}
}
