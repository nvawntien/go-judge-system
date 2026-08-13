package user

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/user"

	"github.com/gin-gonic/gin"
)

type GetPublicProfileStatsHandler struct {
	uc inbound.GetPublicProfileStatsUseCase
}

func NewGetPublicProfileStatsHandler(uc inbound.GetPublicProfileStatsUseCase) *GetPublicProfileStatsHandler {
	return &GetPublicProfileStatsHandler{uc: uc}
}

func (h *GetPublicProfileStatsHandler) Handle(c *gin.Context) {
	response.HandleWithParams(c, h.uc.Execute, response.CodeSuccess)
}
