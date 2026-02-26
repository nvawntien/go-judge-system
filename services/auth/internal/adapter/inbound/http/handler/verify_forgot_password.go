package handler

import (
	"go-judge-system/services/auth/internal/adapter/inbound/http/response"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/domain"
	"net/http"

	"github.com/gin-gonic/gin"
)

type VerifyForgotPasswordHandler struct {
	uc inbound.VerifyForgotPasswordUseCase
}

func NewVerifyForgotPasswordHandler(uc inbound.VerifyForgotPasswordUseCase) *VerifyForgotPasswordHandler {
	return &VerifyForgotPasswordHandler{
		uc: uc,
	}
}

func (h *VerifyForgotPasswordHandler) Handle(c *gin.Context) {
	var req dto.VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request payload")
		return
	}

	token, err := h.uc.Execute(c.Request.Context(), req)
	if err != nil {
		switch err {
		case domain.ErrOTPInvalid:
			response.Error(c, http.StatusBadRequest, err.Error())
		case domain.ErrUserInactive:
			response.Error(c, http.StatusForbidden, err.Error())
		case domain.ErrUserNotFound:
			response.Error(c, http.StatusNotFound, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	response.SuccessWithMessage(c, http.StatusOK, "verification successful", gin.H{"reset_token": token})
}
