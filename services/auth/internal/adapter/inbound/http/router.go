package http

import (
	"go-judge-system/services/auth/internal/adapter/inbound/http/handler"

	"github.com/gin-gonic/gin"
)

type Router struct {
	engine     *gin.Engine
	auth       *handler.AuthHandler
	middleware gin.HandlerFunc
}

func NewRouter(authHandler *handler.AuthHandler, authMiddleware gin.HandlerFunc) *Router {
	r := gin.Default()
	return &Router{
		engine:     r,
		auth:       authHandler,
		middleware: authMiddleware,
	}
}

func (r *Router) SetupRoutes() {
	// Health check — used by Docker HEALTHCHECK / K8s probes
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	auth := r.engine.Group("/api/v1/auth")
	{
		auth.POST("/register", r.auth.Register.Handle)
		auth.POST("/login", r.auth.Login.Handle)
		auth.POST("/logout", r.middleware, r.auth.Logout.Handle)
		
		email := auth.Group("/email")
		{
			email.POST("/verify", r.auth.VerifyEmail.Handle)
			email.POST("/resend-verification", r.auth.ResendVerification.Handle)
		}

		password := auth.Group("/password")
		{
			password.POST("/forgot", r.auth.ForgotPassword.Handle)
		}
	}
}

func (r *Router) Start(port string) error {
	return r.engine.Run(":" + port)
}
