package user

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/user"

	"github.com/gin-gonic/gin"
)

type GetMyProfileStatsHandler struct {
	uc inbound.GetMyProfileStatsUseCase
}

func NewGetMyProfileStatsHandler(uc inbound.GetMyProfileStatsUseCase) *GetMyProfileStatsHandler {
	return &GetMyProfileStatsHandler{uc: uc}
}

func (h *GetMyProfileStatsHandler) Handle(c *gin.Context) {
	response.HandleWithClaimsNoBody(c, h.uc.Execute, response.CodeSuccess)
}
