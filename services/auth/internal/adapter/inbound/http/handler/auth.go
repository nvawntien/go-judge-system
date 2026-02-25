package handler

type AuthHandler struct {
	RegisterHandler *RegisterHandler
	VerifyOTPHandler   *VerifyOTPHandler
	ResendOTPHandler   *ResendOTPHandler
}

func NewAuthHandler(registerHandler *RegisterHandler, verifyOTPHandler *VerifyOTPHandler, resendOTPHandler *ResendOTPHandler) *AuthHandler {
	return &AuthHandler{
		RegisterHandler:  registerHandler,
		VerifyOTPHandler: verifyOTPHandler,
		ResendOTPHandler: resendOTPHandler,
	}
}
