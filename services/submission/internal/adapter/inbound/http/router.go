package http

import (
	"go-judge-system/services/submission/internal/adapter/inbound/http/handler"

	"github.com/gin-gonic/gin"
)

type Router struct {
	engine            *gin.Engine
	submissionHandler *handler.SubmissionHandler
	authMiddleware    gin.HandlerFunc
}

func NewRouter(submissionHandler *handler.SubmissionHandler, authMiddleware gin.HandlerFunc) *Router {
	return &Router{
		engine:            gin.Default(),
		submissionHandler: submissionHandler,
		authMiddleware:    authMiddleware,
	}
}

func (r *Router) SetupRoutes() {
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1 := r.engine.Group("/api/v1")
	auth := v1.Group("")
	auth.Use(r.authMiddleware)
	{
		auth.POST("/submissions", r.submissionHandler.CreateSubmission.Handle)
	}

	my := v1.Group("/my")
	my.Use(r.authMiddleware)
	{
		my.GET("/submissions", r.submissionHandler.ListMySubmissions.Handle)
		my.GET("/submissions/:id", r.submissionHandler.GetMySubmission.Handle)
	}

	admin := v1.Group("/admin")
	admin.Use(r.authMiddleware)
}

func (r *Router) Start(port string) error {
	return r.engine.Run(":" + port)
}
