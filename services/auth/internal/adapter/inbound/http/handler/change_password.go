package handler

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type ChangePasswordHandler struct {
	uc inbound.ChangePasswordUseCase
}

func NewChangePasswordHandler(uc inbound.ChangePasswordUseCase) *ChangePasswordHandler {
	return &ChangePasswordHandler{uc: uc}
}

func (h *ChangePasswordHandler) Handle(c *gin.Context) {
	response.HandleVoidWithClaims(c, h.uc.Execute, response.CodeSuccess, "password changed successfully")
}
