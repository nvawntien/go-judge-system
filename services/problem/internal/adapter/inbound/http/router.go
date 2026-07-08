package http

import (
	pkgmiddleware "go-judge-system/pkg/middleware"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/problem/internal/adapter/inbound/http/handler"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Router struct {
	engine         *gin.Engine
	userHandler    *handler.UserHandler
	adminHandler   *handler.AdminHandler
	authMiddleware gin.HandlerFunc
}

func NewRouter(
	userHandler *handler.UserHandler,
	adminHandler *handler.AdminHandler,
	authMiddleware gin.HandlerFunc,
	logger *zap.Logger,
) *Router {
	r := gin.New()
	r.Use(pkgmiddleware.Recovery(logger))
	r.Use(pkgmiddleware.UnifiedLogger(logger))

	return &Router{
		engine:         r,
		userHandler:    userHandler,
		adminHandler:   adminHandler,
		authMiddleware: authMiddleware,
	}
}

func (r *Router) SetupRoutes() {
	// Health check — used by Docker HEALTHCHECK / K8s probes
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1 := r.engine.Group("/api/v1")
	isContributor := pkgmiddleware.RequireRole(rbac.RoleContributor)
	isModerator := pkgmiddleware.RequireRole(rbac.RoleModerator)

	v1.GET("/problems", r.userHandler.ListProblems.Handle)
	v1.GET("/problems/:slug", r.userHandler.GetProblem.Handle)
	v1.GET("/tags", r.userHandler.ListTags.Handle)

	// Admin routes
	admin := v1.Group("/admin")
	admin.Use(r.authMiddleware)
	{
		// Problem management
		admin.GET("/problems", isContributor, r.adminHandler.ListProblems.Handle)
		admin.GET("/problems/:problem_id", isContributor, r.adminHandler.GetProblem.Handle)
		admin.POST("/problems", isContributor, r.adminHandler.CreateProblem.Handle)
		admin.PUT("/problems/:problem_id", isContributor, r.adminHandler.UpdateProblem.Handle)
		admin.PATCH("/problems/:problem_id/publish", isModerator, r.adminHandler.PublishProblem.Handle)
		admin.PATCH("/problems/:problem_id/hidden", isModerator, r.adminHandler.HiddenProblem.Handle)
		// tag management
		admin.GET("/tags", isContributor, r.adminHandler.ListTags.Handle)
		admin.POST("/tags", isModerator, r.adminHandler.CreateTag.Handle)
		admin.PUT("/tags/:tag_id", isModerator, r.adminHandler.UpdateTag.Handle)
		admin.DELETE("/tags/:tag_id", isModerator, r.adminHandler.DeleteTag.Handle)
		// test case management
		admin.POST("/problems/:problem_id/testcases", isContributor, r.adminHandler.UploadTestCase.Handle)
	}
}

func (r *Router) Start(port string) error {
	return r.engine.Run(":" + port)
}
