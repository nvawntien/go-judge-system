package handler

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type VerifyActivationHandler struct {
	uc inbound.VerifyActivationUseCase
}

func NewVerifyActivationHandler(uc inbound.VerifyActivationUseCase) *VerifyActivationHandler {
	return &VerifyActivationHandler{
		uc: uc,
	}
}

func (h *VerifyActivationHandler) Handle(c *gin.Context) {
	response.HandleVoid(c, h.uc.Execute, response.CodeSuccess, "verification successful, your account is now active")
}
