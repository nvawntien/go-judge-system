package auth

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type ResendVerificationHandler struct {
	uc inbound.ResendVerificationUseCase
}

func NewResendVerificationHandler(uc inbound.ResendVerificationUseCase) *ResendVerificationHandler {
	return &ResendVerificationHandler{uc: uc}
}

func (h *ResendVerificationHandler) Handle(c *gin.Context) {
	response.HandleVoid(
		c,
		h.uc.Execute,
		response.CodeSuccess,
		"If the email is valid, a link has been sent. Please check your email.",
	)
}
