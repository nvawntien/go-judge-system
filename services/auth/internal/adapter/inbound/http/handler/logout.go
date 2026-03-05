package handler

import (
	"go-judge-system/pkg/response"

	"github.com/gin-gonic/gin"
)

type LogoutHandler struct{}

func NewLogoutHandler() *LogoutHandler {
	return &LogoutHandler{}
}

func (h *LogoutHandler) Handle(c *gin.Context) {
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	response.SuccessWithMessage(c, response.CodeSuccess, "logged out successfully", nil)
}
