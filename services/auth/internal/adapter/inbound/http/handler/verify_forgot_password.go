package handler

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type VerifyForgotPasswordHandler struct {
	uc inbound.VerifyForgotPasswordUseCase
}

func NewVerifyForgotPasswordHandler(uc inbound.VerifyForgotPasswordUseCase) *VerifyForgotPasswordHandler {
	return &VerifyForgotPasswordHandler{
		uc: uc,
	}
}

func (h *VerifyForgotPasswordHandler) Handle(c *gin.Context) {
	response.HandleWithMessage(c, h.uc.Execute, response.CodeSuccess, "verify forgot password successfully")
}
