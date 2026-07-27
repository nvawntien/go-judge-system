package container

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go-judge-system/pkg/config"
	grpcin "go-judge-system/workers/judge/internal/adapter/inbound/grpc"
	kafkain "go-judge-system/workers/judge/internal/adapter/inbound/kafka"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type App struct {
	Config        *config.Config
	JobConsumer   *kafkain.JudgeJobConsumer
	GRPC          *grpcin.Server
	Logger        *zap.Logger
	KafkaProducer sarama.SyncProducer
}

func NewApp(
	cfg *config.Config,
	jobConsumer *kafkain.JudgeJobConsumer,
	grpcServer *grpcin.Server,
	logger *zap.Logger,
	producer sarama.SyncProducer,
) *App {
	return &App{
		Config:        cfg,
		JobConsumer:   jobConsumer,
		GRPC:          grpcServer,
		Logger:        logger,
		KafkaProducer: producer,
	}
}

func (a *App) Run() error {
	a.Logger.Info("Starting Judge Worker Service")

	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()

	consumerErrCh := make(chan error, 1)
	go func() {
		consumerErrCh <- a.JobConsumer.Run(consumerCtx)
	}()

	grpcErrCh := make(chan error, 1)
	go func() {
		a.Logger.Info("Starting Judge Worker gRPC server", zap.Int("port", a.Config.Server.GRPCPort))
		grpcErrCh <- a.GRPC.Start()
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	select {
	case err := <-consumerErrCh:
		if err == nil {
			err = errors.New("judge job consumer stopped unexpectedly")
		}
		return err
	case err := <-grpcErrCh:
		if err == nil {
			err = errors.New("judge gRPC server stopped unexpectedly")
		}
		return err
	case sig := <-signalCh:
		a.Logger.Info("shutdown signal received", zap.String("signal", sig.String()))
	}

	consumerCancel()
	a.GRPC.Stop()

	// Wait for consumer to finish with timeout
	consumerDone := make(chan struct{})
	go func() {
		if err := <-consumerErrCh; err != nil && !errors.Is(err, context.Canceled) {
			a.Logger.Warn("consumer error during shutdown", zap.Error(err))
		}
		close(consumerDone)
	}()

	select {
	case <-consumerDone:
		a.Logger.Info("consumer gracefully shutdown")
	case <-time.After(10 * time.Second):
		a.Logger.Warn("consumer shutdown timeout")
	}

	return nil
}

func (a *App) Close() error {
	var closeErr error
	var wg sync.WaitGroup

	if a.KafkaProducer != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := a.KafkaProducer.Close(); err != nil {
				a.Logger.Error("failed to close kafka producer", zap.Error(err))
				closeErr = errors.Join(closeErr, err)
			}
		}()
	}

	if a.JobConsumer != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := a.JobConsumer.Close(); err != nil {
				a.Logger.Error("failed to close judge job consumer", zap.Error(err))
				closeErr = errors.Join(closeErr, err)
			}
		}()
	}

	if a.GRPC != nil {
		a.GRPC.Stop()
	}

	if a.Logger != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := a.Logger.Sync(); err != nil {
				closeErr = errors.Join(closeErr, err)
			}
		}()
	}

	wg.Wait()
	return closeErr
}
