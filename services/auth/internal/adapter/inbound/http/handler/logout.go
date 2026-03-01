package handler

import (
	"go-judge-system/services/auth/internal/adapter/inbound/http/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

type LogoutHandler struct{}

func NewLogoutHandler() *LogoutHandler {
	return &LogoutHandler{}
}

func (h *LogoutHandler) Handle(c *gin.Context) {
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	response.SuccessWithMessage(c, http.StatusOK, "logged out successfully", nil)
}
