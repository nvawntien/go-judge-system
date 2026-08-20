package auth

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type RegisterHandler struct {
	uc inbound.RegisterUseCase
}

func NewRegisterHandler(uc inbound.RegisterUseCase) *RegisterHandler {
	return &RegisterHandler{uc: uc}
}

func (h *RegisterHandler) Handle(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleError(c, response.NewAppError(response.CodeBadRequest, "invalid request payload", err))
		return
	}
	req.ClientIP = clientIP(c)
	if err := h.uc.Execute(c.Request.Context(), req); err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessWithMessage(c, response.CodeCreated, "registration successful, please check your email to verify your account", nil)
}
