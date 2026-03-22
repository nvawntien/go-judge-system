package container

import (
	"errors"
	"fmt"
	"go-judge-system/pkg/config"
	"go-judge-system/services/submission/internal/adapter/inbound/http"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type App struct {
	Config        *config.Config
	Database      *gorm.DB
	Router        *http.Router
	Logger        *zap.Logger
	KafkaProducer sarama.SyncProducer
}

func NewApp(cfg *config.Config, database *gorm.DB, router *http.Router, logger *zap.Logger, producer sarama.SyncProducer) *App {
	return &App{Config: cfg, Database: database, Router: router, Logger: logger, KafkaProducer: producer}
}

func (a *App) Run() error {
	a.Router.SetupRoutes()
	port := fmt.Sprintf("%d", a.Config.Server.Port)
	a.Logger.Info("Starting Submission Service", zap.String("port", port))
	return a.Router.Start(port)
}

func (a *App) Close() error {
	var closeErr error

	if a.KafkaProducer != nil {
		if err := a.KafkaProducer.Close(); err != nil {
			a.Logger.Error("failed to close kafka producer", zap.Error(err))
			closeErr = errors.Join(closeErr, err)
		}
	}

	if a.Database != nil {
		sqlDB, err := a.Database.DB()
		if err == nil {
			if err = sqlDB.Close(); err != nil {
				a.Logger.Error("failed to close database connection", zap.Error(err))
				closeErr = errors.Join(closeErr, err)
			}
		}
	}

	if a.Logger != nil {
		if err := a.Logger.Sync(); err != nil {
			closeErr = errors.Join(closeErr, err)
		}
	}

	return closeErr
}
