package auth

import (
	"go-judge-system/pkg/response"
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
	response.HandleVoid(
		c,
		h.uc.Execute,
		response.CodeSuccess,
		"email verified successfully, your account is now active",
	)
}
