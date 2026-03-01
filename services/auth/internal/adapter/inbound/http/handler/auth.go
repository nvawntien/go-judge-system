package handler

type AuthHandler struct {
	RegisterHandler             *RegisterHandler
	VerifyActivationHandler     *VerifyActivationHandler
	ResendOTPHandler            *ResendOTPHandler
	ForgotPasswordHandler       *ForgotPasswordHandler
	VerifyForgotPasswordHandler *VerifyForgotPasswordHandler
	ResetPasswordHandler        *ResetPasswordHandler
	LoginHandler                *LoginHandler
	ChangePasswordHandler       *ChangePasswordHandler
}

func NewAuthHandler(
	registerHandler *RegisterHandler,
	verifyActivationHandler *VerifyActivationHandler,
	resendOTPHandler *ResendOTPHandler,
	forgotPasswordHandler *ForgotPasswordHandler,
	verifyForgotPasswordHandler *VerifyForgotPasswordHandler,
	resetPasswordHandler *ResetPasswordHandler,
	loginHandler *LoginHandler,
	changePasswordHandler *ChangePasswordHandler,
) *AuthHandler {
	return &AuthHandler{
		RegisterHandler:             registerHandler,
		VerifyActivationHandler:     verifyActivationHandler,
		ResendOTPHandler:            resendOTPHandler,
		ForgotPasswordHandler:       forgotPasswordHandler,
		VerifyForgotPasswordHandler: verifyForgotPasswordHandler,
		ResetPasswordHandler:        resetPasswordHandler,
		LoginHandler:                loginHandler,
		ChangePasswordHandler:       changePasswordHandler,
	}
}
