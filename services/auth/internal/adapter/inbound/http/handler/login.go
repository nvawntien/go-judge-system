package handler

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"

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
		response.Error(c, response.CodeBadRequest, "invalid request payload")
		return
	}

	res, err := h.uc.Execute(c.Request.Context(), req)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	c.SetCookie("access_token", res.AccessToken, res.AccessExpire, "/", "", false, true)
	c.SetCookie("refresh_token", res.RefreshToken, res.RefreshExpire, "/", "", false, true)
	response.Success(c, response.CodeSuccess, res)
}
