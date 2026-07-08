package tag

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"

	"github.com/gin-gonic/gin"
)

type DeleteTagHandler struct {
	uc inbound.DeleteTagUseCase
}

func NewDeleteTagHandler(uc inbound.DeleteTagUseCase) *DeleteTagHandler {
	return &DeleteTagHandler{uc: uc}
}

func (h *DeleteTagHandler) Handle(c *gin.Context) {
	response.HandleVoidWithParamsAndClaims(c, h.uc.Execute, response.CodeSuccess, "tag deactivated successfully")
}
