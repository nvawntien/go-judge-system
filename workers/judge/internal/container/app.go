package container

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-judge-system/pkg/config"
	grpcin "go-judge-system/workers/judge/internal/adapter/inbound/grpc"
	kafkain "go-judge-system/workers/judge/internal/adapter/inbound/kafka"
	"go-judge-system/workers/judge/internal/adapter/outbound/execute"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
	googlegrpc "google.golang.org/grpc"
)

type App struct {
	Config          *config.Config
	JobConsumer     *kafkain.JudgeJobConsumer
	GRPC            *grpcin.Server
	Logger          *zap.Logger
	KafkaProducer   sarama.SyncProducer
	ProblemConn     *googlegrpc.ClientConn
	SandboxConn     *SandboxClientConn
	SandboxExecutor *execute.GoJudgeClient
}

func NewApp(
	cfg *config.Config,
	jobConsumer *kafkain.JudgeJobConsumer,
	grpcServer *grpcin.Server,
	logger *zap.Logger,
	producer sarama.SyncProducer,
	problemConn *googlegrpc.ClientConn,
	sandboxConn *SandboxClientConn,
	sandboxExecutor *execute.GoJudgeClient,
) *App {
	return &App{
		Config:          cfg,
		JobConsumer:     jobConsumer,
		GRPC:            grpcServer,
		Logger:          logger,
		KafkaProducer:   producer,
		ProblemConn:     problemConn,
		SandboxConn:     sandboxConn,
		SandboxExecutor: sandboxExecutor,
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
	addCloseErr := func(component string, err error) {
		if err == nil {
			return
		}
		if a.Logger != nil {
			a.Logger.Error("failed to close judge worker dependency", zap.String("component", component), zap.Error(err))
		}
		closeErr = errors.Join(closeErr, err)
	}

	// Stop public RunCode requests before closing the sandbox dependency they
	// may use. ConsumerGroup.Close then prevents further Kafka job intake; it
	// waits for Sarama's active consumption loops before returning.
	if a.GRPC != nil {
		a.GRPC.Stop()
	}
	if a.JobConsumer != nil {
		addCloseErr("judge job consumer", a.JobConsumer.Close())
	}

	// The testcase cache owns best-effort FileDelete lifecycle RPCs. It must
	// finish (or cancel) those before the sandbox ClientConn is closed.
	if a.SandboxExecutor != nil {
		a.SandboxExecutor.Close()
	}
	if a.SandboxConn != nil {
		addCloseErr("go-judge sandbox gRPC connection", a.SandboxConn.ClientConn.Close())
	}

	if a.ProblemConn != nil {
		addCloseErr("problem gRPC connection", a.ProblemConn.Close())
	}

	if a.KafkaProducer != nil {
		addCloseErr("kafka producer", a.KafkaProducer.Close())
	}

	if a.Logger != nil {
		if err := a.Logger.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) {
			closeErr = errors.Join(closeErr, err)
		}
	}

	return closeErr
}
