package http

import (
	"context"
	"net/http"

	pkgmiddleware "go-judge-system/pkg/middleware"
	"go-judge-system/services/submission/internal/adapter/inbound/http/handler"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Router struct {
	engine         *gin.Engine
	userHandler    *handler.UserHandler
	adminHandler   *handler.AdminHandler
	authMiddleware gin.HandlerFunc
	server         *http.Server
}

func NewRouter(
	userHandler *handler.UserHandler,
	adminHandler *handler.AdminHandler,
	authMiddleware gin.HandlerFunc,
	logger *zap.Logger,
) *Router {
	engine := gin.New()
	engine.Use(pkgmiddleware.Recovery(logger))
	engine.Use(pkgmiddleware.UnifiedLogger(logger))

	return &Router{
		engine:         engine,
		userHandler:    userHandler,
		adminHandler:   adminHandler,
		authMiddleware: authMiddleware,
	}
}

func (r *Router) SetupRoutes() {
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1 := r.engine.Group("/api/v1")
	v1.POST("/submissions", r.authMiddleware, r.userHandler.CreateSubmission.Handle)
	v1.GET("/submissions/:submission_id", r.authMiddleware, r.userHandler.GetSubmission.Handle)
	v1.GET("/me/submissions", r.authMiddleware, r.userHandler.ListMySubmissions.Handle)
	v1.GET("/admin/submissions", r.authMiddleware, r.adminHandler.ListSubmissions.Handle)
}

func (r *Router) Start(port string) error {
	r.server = &http.Server{
		Addr:    ":" + port,
		Handler: r.engine,
	}

	return r.server.ListenAndServe()
}

func (r *Router) Shutdown(ctx context.Context) error {
	if r.server == nil {
		return nil
	}
	return r.server.Shutdown(ctx)
}
