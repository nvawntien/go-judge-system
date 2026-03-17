package submission

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type ListSubmissionsHandler struct {
	uc inbound.ListSubmissionsUseCase
}

func NewListSubmissionsHandler(uc inbound.ListSubmissionsUseCase) *ListSubmissionsHandler {
	return &ListSubmissionsHandler{uc: uc}
}

func (h *ListSubmissionsHandler) Handle(c *gin.Context) {
	response.HandleWithQuery(c, h.uc.Execute, response.CodeSuccess)
}

func (h *ListSubmissionsHandler) HandleMy(c *gin.Context) {
	response.HandleWithQueryAndClaims(c, h.uc.ExecuteMy, response.CodeSuccess)
}

func (h *ListSubmissionsHandler) HandleProblem(c *gin.Context) {
	response.HandleWithParamsAndQuery[dto.ProblemIDRequest, dto.ListProblemSubmissionsQueryRequest](c, h.uc.ExecuteProblem, response.CodeSuccess)
}
