package handler

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/port/inbound"

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
		response.Error(c, response.CodeUnauthorized, "missing refresh token")
		return
	}

	res, err := h.uc.Execute(c.Request.Context(), refreshToken)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	c.SetCookie("access_token", res.AccessToken, res.AccessExpire, "/", "", false, true)
	c.SetCookie("refresh_token", res.RefreshToken, res.RefreshExpire, "/", "", false, true)
	response.Success(c, response.CodeSuccess, res)
}
