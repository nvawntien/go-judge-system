package container

import (
	"context"
	"errors"
	"fmt"
	"go-judge-system/pkg/config"
	"go-judge-system/services/submission/internal/adapter/inbound/http"
	kafkain "go-judge-system/services/submission/internal/adapter/inbound/kafka"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type App struct {
	Config         *config.Config
	Database       *gorm.DB
	Router         *http.Router
	ResultConsumer *kafkain.JudgeResultConsumer
	Logger         *zap.Logger
	KafkaProducer  sarama.SyncProducer
}

func NewApp(
	cfg *config.Config,
	database *gorm.DB,
	router *http.Router,
	resultConsumer *kafkain.JudgeResultConsumer,
	logger *zap.Logger,
	producer sarama.SyncProducer,
) *App {
	return &App{
		Config:         cfg,
		Database:       database,
		Router:         router,
		ResultConsumer: resultConsumer,
		Logger:         logger,
		KafkaProducer:  producer,
	}
}

func (a *App) Run() error {
	a.Router.SetupRoutes()
	port := fmt.Sprintf("%d", a.Config.Server.Port)
	a.Logger.Info("Starting Submission Service", zap.String("port", port))

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- a.Router.Start(port)
	}()

	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()

	consumerErrCh := make(chan error, 1)
	go func() {
		consumerErrCh <- a.ResultConsumer.Run(consumerCtx)
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	select {
	case err := <-serverErrCh:
		if err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			return err
		}
		consumerCancel()
		if consumerErr := <-consumerErrCh; consumerErr != nil {
			return consumerErr
		}
		return nil
	case err := <-consumerErrCh:
		if err == nil {
			err = errors.New("judge result consumer stopped unexpectedly")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if shutdownErr := a.Router.Shutdown(ctx); shutdownErr != nil {
			return errors.Join(err, shutdownErr)
		}

		serverErr := <-serverErrCh
		if serverErr != nil && !errors.Is(serverErr, nethttp.ErrServerClosed) {
			return errors.Join(err, serverErr)
		}

		return err
	case sig := <-signalCh:
		a.Logger.Info("shutdown signal received", zap.String("signal", sig.String()))
	}

	consumerCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.Router.Shutdown(ctx); err != nil {
		return err
	}

	err := <-serverErrCh
	if err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
		return err
	}

	if consumerErr := <-consumerErrCh; consumerErr != nil {
		return consumerErr
	}

	return nil
}

func (a *App) Close() error {
	var closeErr error

	if a.KafkaProducer != nil {
		if err := a.KafkaProducer.Close(); err != nil {
			a.Logger.Error("failed to close kafka producer", zap.Error(err))
			closeErr = errors.Join(closeErr, err)
		}
	}

	if a.ResultConsumer != nil {
		if err := a.ResultConsumer.Close(); err != nil {
			a.Logger.Error("failed to close judge result consumer", zap.Error(err))
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
