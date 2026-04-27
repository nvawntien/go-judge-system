package auth

import (
	"go-judge-system/pkg/response"
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
	response.HandleVoid(
		c,
		h.uc.Execute,
		response.CodeSuccess,
		"If the email is valid, a link has been sent. Please check your email.",
	)
}
