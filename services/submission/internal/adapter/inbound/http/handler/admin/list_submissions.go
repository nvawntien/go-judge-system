package admin

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/admin"

	"github.com/gin-gonic/gin"
)

type ListSubmissionsHandler struct {
	uc inbound.ListAdminSubmissionsUseCase
}

func NewListSubmissionsHandler(
	uc inbound.ListAdminSubmissionsUseCase,
) *ListSubmissionsHandler {
	return &ListSubmissionsHandler{uc: uc}
}

func (h *ListSubmissionsHandler) Handle(c *gin.Context) {
	response.HandleWithQueryAndClaims(
		c,
		h.uc.Execute,
		response.CodeSuccess,
	)
}
