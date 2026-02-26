package handler

import (
	"go-judge-system/services/auth/internal/adapter/inbound/http/response"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/domain"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ForgotPasswordHandler struct {
	uc inbound.ForgotPasswordUseCase
}

func NewForgotPasswordHandler(uc inbound.ForgotPasswordUseCase) *ForgotPasswordHandler {
	return &ForgotPasswordHandler{
		uc: uc,
	}
}

func (h *ForgotPasswordHandler) Handle(ctx *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, "invalid request payload")
		return
	}

	err := h.uc.Execute(ctx.Request.Context(), req)
	if err != nil {
		switch err {
		case domain.ErrUserNotFound:
			response.Error(ctx, http.StatusNotFound, err.Error())
		case domain.ErrUserInactive:
			response.Error(ctx, http.StatusForbidden, err.Error())
		default:
			response.Error(ctx, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	response.SuccessWithMessage(ctx, http.StatusOK, "OTP sent to your email, please check your inbox to verify your account.", nil)
}
