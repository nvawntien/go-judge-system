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
	response.HandleWithClaimsNoBody(c, h.uc.ExecuteMe, response.CodeSuccess)
}

func (h *GetProfileHandler) HandlePublic(c *gin.Context) {
	response.HandleWithParams(c, h.uc.ExecutePublic, response.CodeSuccess)
}
