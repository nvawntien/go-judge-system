package handler

import (
	"net/http"
	"strconv"

	"go-judge-system/services/problem/internal/application/port/outbound"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// InternalTestCaseHandler serves testcases for internal service-to-service calls.
// This endpoint does NOT go through KrakenD and has no auth middleware,
// because it is only accessible within the Docker bridge network (judge_internal).
type InternalTestCaseHandler struct {
	testCaseRepo outbound.TestCaseRepository
	logger       *zap.Logger
}

func NewInternalTestCaseHandler(
	testCaseRepo outbound.TestCaseRepository,
	logger *zap.Logger,
) *InternalTestCaseHandler {
	return &InternalTestCaseHandler{
		testCaseRepo: testCaseRepo,
		logger:       logger,
	}
}

type internalTestCaseResponse struct {
	ID             int64  `json:"id"`
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
	Order          int    `json:"order"`
}

// Handle GET /internal/v1/problems/:id/testcases
func (h *InternalTestCaseHandler) Handle(c *gin.Context) {
	idStr := c.Param("id")
	problemID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid problem id"})
		return
	}

	testCases, err := h.testCaseRepo.GetByProblemID(c.Request.Context(), problemID)
	if err != nil {
		h.logger.Error("failed to fetch testcases for internal API",
			zap.Int64("problem_id", problemID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	items := make([]internalTestCaseResponse, 0, len(testCases))
	for _, tc := range testCases {
		items = append(items, internalTestCaseResponse{
			ID:             tc.ID,
			Input:          tc.Input,
			ExpectedOutput: tc.ExpectedOutput,
			Order:          tc.Order,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"problem_id": problemID,
		"test_cases": items,
		"total":      len(items),
	})
}
