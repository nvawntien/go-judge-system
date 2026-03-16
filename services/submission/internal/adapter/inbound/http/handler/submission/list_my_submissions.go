package submission

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/submission/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type ListMySubmissionsHandler struct {
	uc inbound.ListSubmissionsUseCase
}

func NewListMySubmissionsHandler(uc inbound.ListSubmissionsUseCase) *ListMySubmissionsHandler {
	return &ListMySubmissionsHandler{uc: uc}
}

func (h *ListMySubmissionsHandler) Handle(c *gin.Context) {
	response.HandleWithQueryAndClaims(c, h.uc.ExecuteMy, response.CodeSuccess)
}
