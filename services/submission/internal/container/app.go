package container

import (
	"context"
	"errors"
	"fmt"

	"go-judge-system/pkg/config"
	"go-judge-system/services/submission/internal/adapter/inbound/http"
	"go-judge-system/services/submission/internal/adapter/outbound/outbox"

	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
	googlegrpc "google.golang.org/grpc"
	"gorm.io/gorm"
)

type App struct {
	Config        *config.Config
	Database      *gorm.DB
	Router        *http.Router
	OutboxRelay   *outbox.OutboxRelay
	Logger        *zap.Logger
	KafkaProducer sarama.SyncProducer
	ProblemConn   *googlegrpc.ClientConn
}

func NewApp(
	cfg *config.Config,
	database *gorm.DB,
	router *http.Router,
	outboxRelay *outbox.OutboxRelay,
	logger *zap.Logger,
	producer sarama.SyncProducer,
	problemConn *googlegrpc.ClientConn,
) *App {
	return &App{
		Config:        cfg,
		Database:      database,
		Router:        router,
		OutboxRelay:   outboxRelay,
		Logger:        logger,
		KafkaProducer: producer,
		ProblemConn:   problemConn,
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

	outboxErrCh := make(chan error, 1)
	go func() {
		outboxErrCh <- a.OutboxRelay.Start(workerCtx, 2*time.Second)
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	select {
	case err := <-serverErrCh:
		workerCancel()
		<-outboxErrCh
		if err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			return err
		}
		return nil
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

	if a.ProblemConn != nil {
		if err := a.ProblemConn.Close(); err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("close Problem Service gRPC connection: %w", err))
		}
	}

	if a.KafkaProducer != nil {
		if err := a.KafkaProducer.Close(); err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("close kafka producer: %w", err))
		}
	}

	if a.Database != nil {
		sqlDB, err := a.Database.DB()
		if err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("obtain SQL database handle: %w", err))
		} else if err = sqlDB.Close(); err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("close database: %w", err))
		}
	}

	if a.Logger != nil {
		if err := a.Logger.Sync(); err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("sync logger: %w", err))
		}
	}

	return closeErr
}
