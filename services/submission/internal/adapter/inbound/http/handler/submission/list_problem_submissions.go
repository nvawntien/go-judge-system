package submission

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type ListProblemSubmissionsHandler struct {
	uc inbound.ListSubmissionsUseCase
}

func NewListProblemSubmissionsHandler(uc inbound.ListSubmissionsUseCase) *ListProblemSubmissionsHandler {
	return &ListProblemSubmissionsHandler{uc: uc}
}

func (h *ListProblemSubmissionsHandler) Handle(c *gin.Context) {
	response.HandleWithParamsAndQuery[dto.ProblemIDRequest, dto.ListProblemSubmissionsQueryRequest](c, h.uc.ExecuteProblem, response.CodeSuccess)
}
