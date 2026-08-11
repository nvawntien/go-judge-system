package admin

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type AdminUsersHandler struct {
	uc inbound.AdminUsersUseCase
}

func NewAdminUsersHandler(uc inbound.AdminUsersUseCase) *AdminUsersHandler {
	return &AdminUsersHandler{uc: uc}
}

func (h *AdminUsersHandler) List(c *gin.Context) {
	response.HandleWithQueryAndClaims(c, h.uc.List, response.CodeSuccess)
}

func (h *AdminUsersHandler) Get(c *gin.Context) {
	response.HandleWithParamsAndClaims(c, h.uc.Get, response.CodeSuccess)
}

func (h *AdminUsersHandler) SetSuspension(c *gin.Context) {
	response.HandleWithClaimsParamsAndBody(c, h.uc.SetSuspension, response.CodeSuccess)
}
