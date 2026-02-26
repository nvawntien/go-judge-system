package handler

type AuthHandler struct {
	RegisterHandler *RegisterHandler
	VerifyActivationHandler *VerifyActivationHandler
	ResendOTPHandler   *ResendOTPHandler
	ForgotPasswordHandler *ForgotPasswordHandler
	VerifyForgotPasswordHandler *VerifyForgotPasswordHandler
}

func NewAuthHandler(registerHandler *RegisterHandler, verifyActivationHandler *VerifyActivationHandler, resendOTPHandler *ResendOTPHandler, forgotPasswordHandler *ForgotPasswordHandler, verifyForgotPasswordHandler *VerifyForgotPasswordHandler) *AuthHandler {
	return &AuthHandler{
		RegisterHandler:  registerHandler,
		VerifyActivationHandler: verifyActivationHandler,
		ResendOTPHandler: resendOTPHandler,
		ForgotPasswordHandler: forgotPasswordHandler,
		VerifyForgotPasswordHandler: verifyForgotPasswordHandler,
	}
}
