package auth

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type VerifyEmailHandler struct {
	uc inbound.VerifyEmailUseCase
}

func NewVerifyEmailHandler(uc inbound.VerifyEmailUseCase) *VerifyEmailHandler {
	return &VerifyEmailHandler{uc: uc}
}

func (h *VerifyEmailHandler) Handle(c *gin.Context) {
	var req dto.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleError(c, response.NewAppError(response.CodeBadRequest, "invalid request payload", err))
		return
	}
	req.ClientIP = clientIP(c)
	if err := h.uc.Execute(c.Request.Context(), req); err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessWithMessage(c, response.CodeSuccess, "email verified successfully, your account is now active", nil)
}
