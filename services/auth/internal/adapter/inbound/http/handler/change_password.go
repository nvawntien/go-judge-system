package handler

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/dto"
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
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, response.CodeUnauthorized, "unauthorized user")
		return
	}

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeBadRequest, "invalid request payload")
		return
	}

	if err := h.uc.Execute(c.Request.Context(), userID, req); err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessWithMessage(c, response.CodeSuccess, "password changed successfully", nil)
}
