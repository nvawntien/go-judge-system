package handler

import (
	"go-judge-system/services/auth/internal/adapter/inbound/http/response"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/domain"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ResetPasswordHandler struct {
	uc inbound.ResetPasswordUseCase
}

func NewResetPasswordHandler(uc inbound.ResetPasswordUseCase) *ResetPasswordHandler {
	return &ResetPasswordHandler{uc: uc}
}

func (h *ResetPasswordHandler) Handle(ctx *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, "invalid request payload")
		return
	}

	err := h.uc.Execute(ctx.Request.Context(), req)
	if err != nil {
		switch err {
		case domain.ErrInvalidOrExpiredToken:
			response.Error(ctx, http.StatusBadRequest, err.Error())
		case domain.ErrUserNotFound:
			response.Error(ctx, http.StatusNotFound, err.Error())
		default:
			response.Error(ctx, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	response.SuccessWithMessage(ctx, http.StatusOK, "password reset successful", nil)
}
