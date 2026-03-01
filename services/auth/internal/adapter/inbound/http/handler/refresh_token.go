package handler

import (
	"go-judge-system/services/auth/internal/adapter/inbound/http/response"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"net/http"

	"github.com/gin-gonic/gin"
)

type RefreshTokenHandler struct {
	uc inbound.RefreshTokenUseCase
}

func NewRefreshTokenHandler(uc inbound.RefreshTokenUseCase) *RefreshTokenHandler {
	return &RefreshTokenHandler{uc: uc}
}

func (h *RefreshTokenHandler) Handle(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "missing refresh token")
		return
	}

	res, err := h.uc.Execute(c.Request.Context(), refreshToken)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	c.SetCookie("access_token", res.AccessToken, res.AccessExpire, "/", "", false, true)
	c.SetCookie("refresh_token", res.RefreshToken, res.RefreshExpire, "/", "", false, true)
	response.Success(c, http.StatusOK, res)
}
