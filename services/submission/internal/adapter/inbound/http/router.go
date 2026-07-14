package http

import (
	"context"
	"net/http"

	pkgmiddleware "go-judge-system/pkg/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Router struct {
	engine         *gin.Engine
	authMiddleware gin.HandlerFunc
	server         *http.Server
}

func NewRouter(authMiddleware gin.HandlerFunc, logger *zap.Logger) *Router {
	engine := gin.New()
	engine.Use(pkgmiddleware.Recovery(logger))
	engine.Use(pkgmiddleware.UnifiedLogger(logger))

	return &Router{
		engine:         engine,
		authMiddleware: authMiddleware,
	}
}

func (r *Router) SetupRoutes() {
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
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
