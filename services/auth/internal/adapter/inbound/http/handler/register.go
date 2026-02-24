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

type RegisterHandler struct {
	uc inbound.RegisterUseCase
}

func NewRegisterHandler(uc inbound.RegisterUseCase) *RegisterHandler {
	return &RegisterHandler{uc: uc}
}

func (h *RegisterHandler) Handle(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request payload")
		return
	}

	if err := h.uc.Execute(c.Request.Context(), req); err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidEmail):
			response.Error(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrPasswordTooShort):
			response.Error(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrUserAlreadyExists):
			response.Error(c, http.StatusConflict, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	response.SuccessWithMessage(c, http.StatusOK, "registration successful, please check your email for the OTP", nil)
}
