package handler

import (
	"go-judge-system/services/auth/internal/adapter/inbound/http/response"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/domain"
	"net/http"

	"github.com/gin-gonic/gin"
)

type LoginHandler struct {
	uc inbound.LoginUseCase
}

func NewLoginHandler(uc inbound.LoginUseCase) *LoginHandler {
	return &LoginHandler{uc: uc}
}

func (h *LoginHandler) Handle(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	res, err := h.uc.Execute(c.Request.Context(), req)
	if err != nil {
		switch err {
		case domain.ErrInvalidCredentials:
			response.Error(c, http.StatusUnauthorized, err.Error())
		case domain.ErrUserInactive:
			response.Error(c, http.StatusForbidden, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	c.SetCookie("access_token", res.AccessToken, res.AccessExpire, "/", "", false, true)
	c.SetCookie("refresh_token", res.RefreshToken, res.RefreshExpire, "/", "", false, true)
	response.Success(c, http.StatusOK, res)
}
