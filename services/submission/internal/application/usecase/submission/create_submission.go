package submission

import (
	"context"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/application/port/inbound"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/application/usecase"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"

	"go.uber.org/zap"
)

type createSubmissionUseCase struct {
	txManager      outbound.TransactionManager
	submissionRepo outbound.SubmissionRepository
	judgePublisher outbound.JudgePublisher
	logger         *zap.Logger
}

func NewCreateSubmissionUseCase(
	txManager outbound.TransactionManager,
	submissionRepo outbound.SubmissionRepository,
	judgePublisher outbound.JudgePublisher,
	logger *zap.Logger,
) inbound.CreateSubmissionUseCase {
	return &createSubmissionUseCase{
		txManager:      txManager,
		submissionRepo: submissionRepo,
		judgePublisher: judgePublisher,
		logger:         logger,
	}
}

func (uc *createSubmissionUseCase) Execute(ctx context.Context, claims auth.Claims, req dto.CreateSubmissionRequest) (dto.SubmissionResponse, error) {
	language, ok := entity.ParseLanguage(req.Language)
	if !ok {
		return dto.SubmissionResponse{}, domain.ErrInvalidLanguage
	}

	if req.SourceCode == "" {
		return dto.SubmissionResponse{}, domain.ErrInvalidSourceCode
	}

	sub := entity.NewSubmission(req.ProblemID, req.ProblemName, claims.UserID, claims.Username, language, req.SourceCode)

	err := uc.txManager.ExecuteInTx(ctx, func(txCtx context.Context) error {
		if err := uc.submissionRepo.Create(txCtx, sub); err != nil {
			return err
		}

		if err := uc.judgePublisher.Publish(txCtx, sub); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		uc.logger.Error("failed to create submission or write to outbox", zap.Error(err))
		return dto.SubmissionResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	return usecase.MapSubmissionToResponse(sub), nil
}
