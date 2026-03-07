package handler

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type ResendOTPHandler struct {
	uc inbound.ResendOTPUseCase
}

func NewResendOTPHandler(uc inbound.ResendOTPUseCase) *ResendOTPHandler {
	return &ResendOTPHandler{
		uc: uc,
	}
}

func (h *ResendOTPHandler) Handle(c *gin.Context) {
	response.HandleVoid(c, h.uc.Execute, response.CodeSuccess, "OTP resent successfully, please check your email")
}
