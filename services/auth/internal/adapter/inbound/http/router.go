package http

import (
	"go-judge-system/services/auth/internal/adapter/inbound/http/handler"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	pkgmiddleware "go-judge-system/pkg/middleware"
)

type Router struct {
	engine     *gin.Engine
	auth       *handler.AuthHandler
	user       *handler.UserHandler
	middleware gin.HandlerFunc
}

func NewRouter(authHandler *handler.AuthHandler, userHandler *handler.UserHandler, authMiddleware gin.HandlerFunc, logger *zap.Logger) *Router {
	r := gin.New()
	r.Use(pkgmiddleware.Recovery(logger))
	r.Use(pkgmiddleware.UnifiedLogger(logger))

	return &Router{
		engine:     r,
		auth:       authHandler,
		user:       userHandler,
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
		auth.POST("/logout-all", r.middleware, r.auth.LogoutAll.Handle)
		auth.POST("/refresh-token", r.auth.RefreshToken.Handle)

		email := auth.Group("/email")
		{
			email.POST("/verify", r.auth.VerifyEmail.Handle)
			email.POST("/resend-verification", r.auth.ResendVerification.Handle)
		}

		password := auth.Group("/password")
		{
			password.POST("/forgot", r.auth.ForgotPassword.Handle)
			password.POST("/reset", r.auth.ResetPassword.Handle)
			password.PUT("/change", r.middleware, r.auth.ChangePassword.Handle)
		}
	}

	user := r.engine.Group("/api/v1/users")
	{
		profile := user.Group("/profile")
		{
			profile.GET("/me", r.middleware, r.user.GetMe.Handle)
			profile.GET("/:username", r.user.GetProfile.Handle)
		}

	}
}

func (r *Router) Start(port string) error {
	return r.engine.Run(":" + port)
}
