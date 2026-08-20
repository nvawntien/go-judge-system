package auth

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type ForgotPasswordHandler struct {
	uc inbound.ForgotPasswordUseCase
}

func NewForgotPasswordHandler(uc inbound.ForgotPasswordUseCase) *ForgotPasswordHandler {
	return &ForgotPasswordHandler{uc: uc}
}

func (h *ForgotPasswordHandler) Handle(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleError(c, response.NewAppError(response.CodeBadRequest, "invalid request payload", err))
		return
	}
	req.ClientIP = clientIP(c)
	if err := h.uc.Execute(c.Request.Context(), req); err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessWithMessage(c, response.CodeSuccess, "If the email is valid, a link has been sent. Please check your email.", nil)
}
