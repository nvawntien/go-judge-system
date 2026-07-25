package user

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/user"

	"github.com/gin-gonic/gin"
)

type ListMySubmissionsHandler struct {
	uc inbound.ListMySubmissionsUseCase
}

func NewListMySubmissionsHandler(uc inbound.ListMySubmissionsUseCase) *ListMySubmissionsHandler {
	return &ListMySubmissionsHandler{uc: uc}
}

func (h *ListMySubmissionsHandler) Handle(c *gin.Context) {
	response.HandleWithQueryAndClaims(
		c,
		h.uc.Execute,
		response.CodeSuccess,
	)
}
