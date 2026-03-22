package container

import (
	"context"
	"errors"
	"fmt"
	"go-judge-system/pkg/config"
	"go-judge-system/services/submission/internal/adapter/inbound/http"
	kafkain "go-judge-system/services/submission/internal/adapter/inbound/kafka"
	"go-judge-system/services/submission/internal/adapter/outbound/outbox"
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
	OutboxRelay    *outbox.OutboxRelay
	Logger         *zap.Logger
	KafkaProducer  sarama.SyncProducer
}

func NewApp(
	cfg *config.Config,
	database *gorm.DB,
	router *http.Router,
	resultConsumer *kafkain.JudgeResultConsumer,
	outboxRelay *outbox.OutboxRelay,
	logger *zap.Logger,
	producer sarama.SyncProducer,
) *App {
	return &App{
		Config:         cfg,
		Database:       database,
		Router:         router,
		ResultConsumer: resultConsumer,
		OutboxRelay:    outboxRelay,
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

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	consumerErrCh := make(chan error, 1)
	go func() {
		consumerErrCh <- a.ResultConsumer.Run(workerCtx)
	}()

	outboxErrCh := make(chan error, 1)
	go func() {
		outboxErrCh <- a.OutboxRelay.Start(workerCtx, 2*time.Second)
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	select {
	case err := <-serverErrCh:
		if err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			return err
		}
		workerCancel()
		<-consumerErrCh
		<-outboxErrCh
		return nil
	case err := <-consumerErrCh:
		if err == nil {
			err = errors.New("judge result consumer stopped unexpectedly")
		}
		return a.shutdownGracefully(err, serverErrCh, workerCancel)
	case err := <-outboxErrCh:
		if err == nil {
			err = errors.New("outbox relay stopped unexpectedly")
		}
		return a.shutdownGracefully(err, serverErrCh, workerCancel)
	case sig := <-signalCh:
		a.Logger.Info("shutdown signal received", zap.String("signal", sig.String()))
	}

	workerCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.Router.Shutdown(ctx); err != nil {
		return err
	}

	err := <-serverErrCh
	if err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
		return err
	}

	<-consumerErrCh
	<-outboxErrCh

	return nil
}

func (a *App) shutdownGracefully(cause error, serverErrCh <-chan error, workerCancel context.CancelFunc) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if shutdownErr := a.Router.Shutdown(ctx); shutdownErr != nil {
		return errors.Join(cause, shutdownErr)
	}

	serverErr := <-serverErrCh
	if serverErr != nil && !errors.Is(serverErr, nethttp.ErrServerClosed) {
		return errors.Join(cause, serverErr)
	}

	workerCancel()
	return cause
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
