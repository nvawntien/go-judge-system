package handler

import "go-judge-system/services/auth/internal/adapter/inbound/http/handler/auth"

type AuthHandler struct {
	Register           *auth.RegisterHandler
	VerifyEmail        *auth.VerifyEmailHandler
	ResendVerification *auth.ResendVerificationHandler
	Login              *auth.LoginHandler
	Logout             *auth.LogoutHandler
	ForgotPassword     *auth.ForgotPasswordHandler
	ResetPassword      *auth.ResetPasswordHandler
	ChangePassword     *auth.ChangePasswordHandler
}

func NewAuthHandler(
	registerHandler *auth.RegisterHandler,
	verifyEmailHandler *auth.VerifyEmailHandler,
	resendVerificationHandler *auth.ResendVerificationHandler,
	loginHandler *auth.LoginHandler,
	logoutHandler *auth.LogoutHandler,
	forgotPasswordHandler *auth.ForgotPasswordHandler,
	resetPasswordHandler *auth.ResetPasswordHandler,
	changePasswordHandler *auth.ChangePasswordHandler,

) *AuthHandler {
	return &AuthHandler{
		Register:           registerHandler,
		VerifyEmail:        verifyEmailHandler,
		ResendVerification: resendVerificationHandler,
		Login:              loginHandler,
		Logout:             logoutHandler,
		ForgotPassword:     forgotPasswordHandler,
		ResetPassword:      resetPasswordHandler,
		ChangePassword:     changePasswordHandler,
	}
}
