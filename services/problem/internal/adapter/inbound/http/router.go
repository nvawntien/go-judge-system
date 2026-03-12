package http

import (
	"go-judge-system/services/problem/internal/adapter/inbound/http/handler"

	"github.com/gin-gonic/gin"
)

type Router struct {
	engine          *gin.Engine
	problemHandler  *handler.ProblemHandler
	testcaseHandler *handler.TestCaseHandler
	authMiddleware  gin.HandlerFunc
}

func NewRouter(
	problemHandler *handler.ProblemHandler,
	testcaseHandler *handler.TestCaseHandler,
	authMiddleware gin.HandlerFunc,
) *Router {
	return &Router{
		engine:          gin.Default(),
		problemHandler:  problemHandler,
		testcaseHandler: testcaseHandler,
		authMiddleware:  authMiddleware,
	}
}

func (r *Router) SetupRoutes() {
	// Health check — used by Docker HEALTHCHECK / K8s probes
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1 := r.engine.Group("/api/v1")

	// ---- Public routes (slug-based, user-facing) ----
	problems := v1.Group("/problems")
	{
		problems.GET("", r.problemHandler.ListProblems.Handle)
		problems.GET("/:slug", r.problemHandler.GetProblem.Handle)
	}

	// ---- Authenticated user routes ----
	my := v1.Group("/my")
	my.Use(r.authMiddleware)
	{
		my.GET("/problems", r.problemHandler.ListProblems.HandleMy)
	}

	// ---- Admin routes (id-based, protected) ----
	admin := v1.Group("/admin")
	admin.Use(r.authMiddleware)
	{
		// Problem management
		admin.GET("/problems", r.problemHandler.ListProblems.HandleAdmin)
		admin.GET("/problems/:id", r.problemHandler.GetProblem.HandleAdmin)
		admin.POST("/problems", r.problemHandler.CreateProblem.Handle)
		admin.PUT("/problems/:id", r.problemHandler.UpdateProblem.Handle)
		admin.DELETE("/problems/:id", r.problemHandler.DeleteProblem.Handle)
		admin.PUT("/problems/:id/publish", r.problemHandler.PublishProblem.Handle)
		admin.PUT("/problems/:id/hide", r.problemHandler.HideProblem.Handle)

		// TestCase management (problem-scoped)
		admin.POST("/problems/:id/testcases", r.testcaseHandler.CreateTestCase.Handle)
		admin.GET("/problems/:id/testcases", r.testcaseHandler.ListTestCases.Handle)
		admin.PUT("/testcases/:id", r.testcaseHandler.UpdateTestCase.Handle)
		admin.DELETE("/testcases/:id", r.testcaseHandler.DeleteTestCase.Handle)
	}
}

func (r *Router) Start(port string) error {
	return r.engine.Run(":" + port)
}
