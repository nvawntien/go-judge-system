package handler

import (
	"errors"
	"go-judge-system/services/auth/internal/adapter/inbound/http/response"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/domain"
	"net/http"

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
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.uc.Execute(c.Request.Context(), userID, req); err != nil {
		switch {
		case errors.Is(err, domain.ErrIncorrecOldPassword):
			response.Error(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrPasswordTooShort):
			response.Error(c, http.StatusBadRequest, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	response.SuccessWithMessage(c, http.StatusOK, "password changed successfully", nil)
}
