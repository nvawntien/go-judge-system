package handler

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type GetProfileHandler struct {
	uc inbound.GetProfileUseCase
}

func NewGetProfileHandler(uc inbound.GetProfileUseCase) *GetProfileHandler {
	return &GetProfileHandler{uc: uc}
}

func (h *GetProfileHandler) HandleMe(c *gin.Context) {
	username := c.GetString("username")
	if username == "" {
		response.Error(c, response.CodeUnauthorized, "Unauthorized")
		return
	}

	profile, err := h.uc.Execute(c, username)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, response.CodeSuccess, profile)
}

func (h *GetProfileHandler) HandlePublic(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		response.Error(c, response.CodeParamInvalid, "invalid username")
		return
	}

	profile, err := h.uc.Execute(c, username)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, response.CodeSuccess, profile)
}
