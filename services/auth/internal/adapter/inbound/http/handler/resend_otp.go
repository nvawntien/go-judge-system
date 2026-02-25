package handler

import (
	"go-judge-system/services/auth/internal/adapter/inbound/http/response"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/domain"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ResendOTPHandler struct {
	uc inbound.ResendOTPUseCase
}

func NewResendOTPHandler(uc inbound.ResendOTPUseCase) *ResendOTPHandler {
	return &ResendOTPHandler{
		uc: uc,
	}
}

func (h *ResendOTPHandler) Handle(c *gin.Context) {
	var req dto.ResendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request payload")
		return
	}

	if err := h.uc.Execute(c.Request.Context(), req); err != nil {
		switch err {
		case domain.ErrUserNotFound:
			response.Error(c, http.StatusNotFound, err.Error())
		case domain.ErrUserAlreadyActive:
			response.Error(c, http.StatusConflict, err.Error())
		case domain.ErrUserInactive:
			response.Error(c, http.StatusForbidden, err.Error())
		case domain.ErrRateLimitExceeded:
			response.Error(c, http.StatusTooManyRequests, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "internal server error")
			return
		}
		return
	}

	response.SuccessWithMessage(c, http.StatusOK, "OTP resent successfully, please check your email", nil)
}
