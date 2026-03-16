package http

import "github.com/gin-gonic/gin"

type Router struct {
	engine         *gin.Engine
	authMiddleware gin.HandlerFunc
}

func NewRouter(authMiddleware gin.HandlerFunc) *Router {
	return &Router{
		engine:         gin.Default(),
		authMiddleware: authMiddleware,
	}
}

func (r *Router) SetupRoutes() {
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1 := r.engine.Group("/api/v1")

	my := v1.Group("/my")
	my.Use(r.authMiddleware)

	admin := v1.Group("/admin")
	admin.Use(r.authMiddleware)
}

func (r *Router) Start(port string) error {
	return r.engine.Run(":" + port)
}
