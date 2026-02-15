package main

import (
	"fmt"
	"go-judge-system/pkg/cache"
	"go-judge-system/pkg/config"
	"go-judge-system/pkg/database"
	"go-judge-system/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.LoadConfig("/app/config")
	if err != nil {
		panic(err)
	}

	logger, err := logger.NewLogger(cfg.Server.Mode)
	if err != nil {
		panic(err)
	}

	logger.Info("Auth service started", zap.Int("port", cfg.Server.Port))

	db, err := database.ConnectDatabase(cfg.Database)
	if err != nil {
		logger.Error("Failed to connect to database", zap.Error(err))
		panic(err)
	}

	rdb, err := cache.ConnectRedis(cfg.Redis)
	if err != nil {
		logger.Error("Failed to connect to redis", zap.Error(err))
		panic(err)
	}

	_ = db
	_ = rdb
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	r.Run(fmt.Sprintf(":%d", cfg.Server.Port))
}
