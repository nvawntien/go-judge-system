package user

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

func (h *GetProfileHandler) Handle(c *gin.Context) {
	response.HandleWithParams(
		c,
		h.uc.Execute,
		response.CodeSuccess,
	)
}
