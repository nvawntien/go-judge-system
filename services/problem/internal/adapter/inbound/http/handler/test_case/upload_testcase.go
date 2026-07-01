package testcase

import (
	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/response"
	"go-judge-system/services/problem/internal/application/dto"
	"go-judge-system/services/problem/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type UploadTestCaseHandler struct {
	uc inbound.UploadTestCaseUseCase
}

func NewUploadTestCaseHandler(uc inbound.UploadTestCaseUseCase) *UploadTestCaseHandler {
	return &UploadTestCaseHandler{uc: uc}
}

func (h *UploadTestCaseHandler) Handle(c *gin.Context) {
	claims, ok := auth.GetClaims(c)
	if !ok {
		response.HandleError(c, response.NewAppError(response.CodeUnauthorized, "unauthorized", nil))
		return
	}

	var params dto.ProblemIDRequest
	if err := c.ShouldBindUri(&params); err != nil {
		response.HandleError(c, response.NewAppError(response.CodeParamInvalid, "invalid uri params", err))
		return
	}

	var form dto.UploadTestCaseRequest
	if err := c.ShouldBind(&form); err != nil {
		response.HandleError(c, response.NewAppError(response.CodeBadRequest, "invalid form data", err))
		return
	}

	res, err := h.uc.Execute(c.Request.Context(), claims, params, form)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, response.CodeCreated, res)
}
