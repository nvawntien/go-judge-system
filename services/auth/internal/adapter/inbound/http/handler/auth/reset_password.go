package auth

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type ResetPasswordHandler struct {
	uc inbound.ResetPasswordUseCase
}

func NewResetPasswordHandler(uc inbound.ResetPasswordUseCase) *ResetPasswordHandler {
	return &ResetPasswordHandler{uc: uc}
}

func (h *ResetPasswordHandler) Handle(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleError(c, response.NewAppError(response.CodeBadRequest, "invalid request payload", err))
		return
	}
	req.ClientIP = clientIP(c)
	if err := h.uc.Execute(c.Request.Context(), req); err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessWithMessage(c, response.CodeSuccess, "password reset successfully, you can now log in with your new password", nil)
}
