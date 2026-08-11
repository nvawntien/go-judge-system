package container

import (
	"fmt"
	"strings"
	"time"

	auth "go-judge-system/pkg/auth"
	"go-judge-system/pkg/cache"
	"go-judge-system/pkg/config"
	"go-judge-system/pkg/database"
	sharedgrpc "go-judge-system/pkg/grpc"
	"go-judge-system/pkg/kafka"
	"go-judge-system/pkg/logger"
	"go-judge-system/pkg/middleware"
	judgev1 "go-judge-system/pkg/pb/judge/v1"
	problemv1 "go-judge-system/pkg/pb/problem/v1"
	"go-judge-system/services/submission/internal/adapter/inbound/http"
	"go-judge-system/services/submission/internal/adapter/inbound/http/handler"
	adminhandler "go-judge-system/services/submission/internal/adapter/inbound/http/handler/admin"
	userhandler "go-judge-system/services/submission/internal/adapter/inbound/http/handler/user"
	resultconsumer "go-judge-system/services/submission/internal/adapter/inbound/kafka"
	attemptid "go-judge-system/services/submission/internal/adapter/outbound/id"
	judgepublisher "go-judge-system/services/submission/internal/adapter/outbound/judge"
	"go-judge-system/services/submission/internal/adapter/outbound/outbox"
	"go-judge-system/services/submission/internal/adapter/outbound/persistence/postgres"
	problemreader "go-judge-system/services/submission/internal/adapter/outbound/problem"
	streamadapter "go-judge-system/services/submission/internal/adapter/outbound/stream"
	"go-judge-system/services/submission/internal/application/dto"
	admininbound "go-judge-system/services/submission/internal/application/port/inbound/admin"
	resultinbound "go-judge-system/services/submission/internal/application/port/inbound/result"
	userinbound "go-judge-system/services/submission/internal/application/port/inbound/user"
	"go-judge-system/services/submission/internal/application/port/outbound"
	adminusecase "go-judge-system/services/submission/internal/application/usecase/admin"
	resultusecase "go-judge-system/services/submission/internal/application/usecase/result"
	userusecase "go-judge-system/services/submission/internal/application/usecase/user"

	"github.com/google/wire"
	"go.uber.org/zap"
	googlegrpc "google.golang.org/grpc"
)

func ProvideProblemGRPCConfig(cfg *config.Config) (config.ProblemGRPCConfig, error) {
	problemCfg := cfg.ProblemGRPC
	problemCfg.Address = strings.TrimSpace(problemCfg.Address)
	if problemCfg.Address == "" {
		problemCfg.Address = "problem-service:9092"
	}
	if problemCfg.Timeout == 0 {
		problemCfg.Timeout = time.Second
	}
	if problemCfg.Timeout < 0 {
		return config.ProblemGRPCConfig{}, fmt.Errorf("problem gRPC timeout must be greater than zero")
	}

	return problemCfg, nil
}

func ProvideProblemClientConn(cfg config.ProblemGRPCConfig) (*googlegrpc.ClientConn, error) {
	conn, err := sharedgrpc.NewClientConn(cfg.Address, sharedgrpc.WithInsecureTransport())
	if err != nil {
		return nil, fmt.Errorf("create Problem Service gRPC connection: %w", err)
	}
	return conn, nil
}

func ProvideProblemServiceClient(conn *googlegrpc.ClientConn) problemv1.ProblemServiceClient {
	return problemv1.NewProblemServiceClient(conn)
}

func ProvideProblemGRPCTimeout(cfg config.ProblemGRPCConfig) time.Duration {
	return cfg.Timeout
}

type JudgeClientConn struct {
	*googlegrpc.ClientConn
}

func ProvideJudgeGRPCConfig(cfg *config.Config) (config.JudgeGRPCConfig, error) {
	judgeCfg := cfg.JudgeGRPC
	judgeCfg.Address = strings.TrimSpace(judgeCfg.Address)
	if judgeCfg.Address == "" {
		judgeCfg.Address = "judge-worker:9093"
	}
	if judgeCfg.Timeout == 0 {
		judgeCfg.Timeout = 30 * time.Second
	}
	if judgeCfg.Timeout < 0 {
		return config.JudgeGRPCConfig{}, fmt.Errorf("judge gRPC timeout must be greater than zero")
	}
	return judgeCfg, nil
}

func ProvideJudgeClientConn(cfg config.JudgeGRPCConfig) (JudgeClientConn, error) {
	conn, err := sharedgrpc.NewClientConn(cfg.Address, sharedgrpc.WithInsecureTransport())
	if err != nil {
		return JudgeClientConn{}, fmt.Errorf("create Judge Worker gRPC connection: %w", err)
	}
	return JudgeClientConn{ClientConn: conn}, nil
}

func ProvideJudgeServiceClient(conn JudgeClientConn) judgev1.JudgeServiceClient {
	return judgev1.NewJudgeServiceClient(conn.ClientConn)
}

func ProvideRunCodeLimits(cfg *config.Config) dto.RunCodeLimits {
	runCfg := cfg.RunCode
	return dto.RunCodeLimits{
		MaxTestCases:           runCfg.MaxTestCases,
		MaxSourceCodeBytes:     runCfg.MaxSourceCodeBytes,
		MaxStdinBytes:          runCfg.MaxStdinBytes,
		MaxExpectedOutputBytes: runCfg.MaxExpectedOutputBytes,
		MaxCapturedOutputBytes: runCfg.MaxCapturedOutputBytes,
		RequestTimeout:         runCfg.RequestTimeout,
		DefaultTimeLimit:       runCfg.DefaultTimeLimit,
		DefaultMemoryLimitKB:   runCfg.DefaultMemoryLimitKB,
		DefaultOutputLimit:     runCfg.DefaultOutputLimitBytes,
	}
}

func ProvideSSEConfig(cfg *config.Config) (config.SSEConfig, error) {
	sseCfg := cfg.SSE
	sseCfg.TicketSecret = strings.TrimSpace(sseCfg.TicketSecret)
	sseCfg.AllowedOrigin = strings.TrimSpace(sseCfg.AllowedOrigin)
	if sseCfg.TicketSecret == "" {
		return config.SSEConfig{}, fmt.Errorf("sse ticket secret is required")
	}
	if sseCfg.TicketTTL <= 0 {
		return config.SSEConfig{}, fmt.Errorf("sse ticket ttl must be greater than zero")
	}
	if sseCfg.HeartbeatInterval <= 0 {
		return config.SSEConfig{}, fmt.Errorf("sse heartbeat interval must be greater than zero")
	}
	if sseCfg.HeartbeatInterval >= 75*time.Second {
		return config.SSEConfig{}, fmt.Errorf("sse heartbeat interval must be less than edge proxy read timeout")
	}
	if sseCfg.AllowedOrigin == "" {
		sseCfg.AllowedOrigin = "http://localhost:3000"
	}
	return sseCfg, nil
}

func ProvideGetAdminSubmissionDetailUseCase(
	submissionRepo outbound.SubmissionRepository,
	resultRepo outbound.SubmissionResultRepository,
	attemptRepo outbound.SubmissionAttemptRepository,
) admininbound.GetAdminSubmissionDetailUseCase {
	return adminusecase.NewGetAdminSubmissionDetailUseCase(submissionRepo, resultRepo, attemptRepo)
}

func ProvideApplyJudgeResultUseCase(
	submissionRepo outbound.SubmissionRepository,
	resultRepo outbound.SubmissionResultRepository,
	txManager outbound.TransactionManager,
	eventHub outbound.SubmissionEventHub,
	logger *zap.Logger,
	attemptRepo outbound.SubmissionAttemptRepository,
) resultinbound.ApplyJudgeResultUseCase {
	return resultusecase.NewApplyJudgeResultUseCase(submissionRepo, resultRepo, txManager, eventHub, logger, attemptRepo)
}

func ProvideCreateSubmissionUseCase(
	submissionRepo outbound.SubmissionRepository,
	txManager outbound.TransactionManager,
	judgePublisher outbound.JudgePublisher,
	attemptIDs outbound.AttemptIDGenerator,
	problemReader outbound.ProblemReader,
	attemptRepo outbound.SubmissionAttemptRepository,
) userinbound.CreateSubmissionUseCase {
	return userusecase.NewCreateSubmissionUseCase(submissionRepo, txManager, judgePublisher, attemptIDs, problemReader, attemptRepo)
}

func ProvideAdminHandler(
	listSubmissions *adminhandler.ListSubmissionsHandler,
	getSubmissionDetail *adminhandler.GetSubmissionDetailHandler,
	rejudgeSubmission *adminhandler.RejudgeSubmissionHandler,
) *handler.AdminHandler {
	return handler.NewAdminHandler(listSubmissions, getSubmissionDetail, rejudgeSubmission)
}

var InfrastructureProviderSet = wire.NewSet(
	database.ConnectDatabase,
	cache.ConnectRedis,
	logger.NewLogger,
	kafka.NewSyncProducer,
	kafka.NewConsumerGroup,
	ProvideProblemGRPCConfig,
	ProvideProblemClientConn,
	ProvideProblemServiceClient,
	ProvideProblemGRPCTimeout,
	ProvideJudgeGRPCConfig,
	ProvideJudgeClientConn,
	ProvideJudgeServiceClient,
	ProvideRunCodeLimits,
	ProvideSSEConfig,
)

var MiddlewareProviderSet = wire.NewSet(
	middleware.NewAuthMiddleware,
)

var OutboundProviderSet = wire.NewSet(
	postgres.NewSubmissionRepository,
	postgres.NewSubmissionStreamSnapshotRepository,
	postgres.NewSubmissionResultRepository,
	postgres.NewSubmissionAttemptRepository,
	postgres.NewTransactionManager,
	postgres.NewOutboxRepository,
	attemptid.NewUUIDAttemptIDGenerator,
	judgepublisher.NewOutboxJudgePublisher,
	auth.NewRedisLogoutAllIATStore,
	outbox.NewOutboxRelay,
	resultconsumer.NewDLTPublisher,
	problemreader.NewGRPCProblemReader,
	judgepublisher.NewGRPCRunner,
	streamadapter.NewHMACSubmissionStreamTicketService,
	streamadapter.NewSubmissionEventHub,
)

var UseCaseProviderSet = wire.NewSet(
	adminusecase.NewListAdminSubmissionsUseCase,
	ProvideGetAdminSubmissionDetailUseCase,
	adminusecase.NewRejudgeAdminSubmissionUseCase,
	ProvideApplyJudgeResultUseCase,
	ProvideCreateSubmissionUseCase,
	userusecase.NewRunCodeUseCase,
	userusecase.NewGetSubmissionUseCase,
	userusecase.NewListMySubmissionsUseCase,
	userusecase.NewIssueSubmissionStreamTicketUseCase,
)

var InboundProviderSet = wire.NewSet(
	adminhandler.NewListSubmissionsHandler,
	adminhandler.NewGetSubmissionDetailHandler,
	adminhandler.NewRejudgeSubmissionHandler,
	userhandler.NewCreateSubmissionHandler,
	userhandler.NewRunCodeHandler,
	userhandler.NewGetSubmissionHandler,
	userhandler.NewListMySubmissionsHandler,
	userhandler.NewIssueSubmissionStreamTicketHandler,
	userhandler.NewSubmissionEventsHandler,
	ProvideAdminHandler,
	handler.NewUserHandler,
	http.NewRouter,
	resultconsumer.NewJudgeResultConsumer,
)
