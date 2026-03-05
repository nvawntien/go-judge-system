package handler

import (
	"go-judge-system/pkg/response"
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
	response.HandleVoid(
		c,
		h.uc.Execute,
		response.CodeSuccess,
		"password reset successful",
	)
}
