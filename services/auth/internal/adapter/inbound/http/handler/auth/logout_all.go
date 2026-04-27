package auth

import (
	pkgauth "go-judge-system/pkg/auth"
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type LogoutAllHandler struct {
	uc inbound.LogoutAllUseCase
}

func NewLogoutAllHandler(uc inbound.LogoutAllUseCase) *LogoutAllHandler {
	return &LogoutAllHandler{uc: uc}
}

func (h *LogoutAllHandler) Handle(c *gin.Context) {
	claims, ok := pkgauth.GetClaims(c)
	if !ok {
		response.HandleError(c, response.NewAppError(response.CodeUnauthorized, "unauthorized", nil))
		return
	}

	if err := h.uc.Execute(c.Request.Context(), claims.UserID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	response.SuccessWithMessage(c, response.CodeSuccess, "logged out from all devices successfully", nil)
}
