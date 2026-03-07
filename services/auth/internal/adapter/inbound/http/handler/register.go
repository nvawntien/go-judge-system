package handler

import (
	"go-judge-system/pkg/response"
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
	response.HandleVoid(c, h.uc.Execute, response.CodeCreated, "registration successful, please check your email for the OTP")
}
