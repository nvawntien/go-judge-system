package user

import (
	"context"
	"strings"
	"time"

	"go-judge-system/pkg/auth"
	pkgjudge "go-judge-system/pkg/judge"
	"go-judge-system/pkg/response"
	"go-judge-system/services/submission/internal/application/dto"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/user"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"
)

type createSubmissionUseCase struct {
	submissionRepo outbound.SubmissionRepository
	attemptRepo    outbound.SubmissionAttemptRepository
	txManager      outbound.TransactionManager
	judgePublisher outbound.JudgePublisher
	attemptIDs     outbound.AttemptIDGenerator
	problemReader  outbound.ProblemReader
}

func NewCreateSubmissionUseCase(
	submissionRepo outbound.SubmissionRepository,
	txManager outbound.TransactionManager,
	judgePublisher outbound.JudgePublisher,
	attemptIDs outbound.AttemptIDGenerator,
	problemReader outbound.ProblemReader,
	attemptRepos ...outbound.SubmissionAttemptRepository,
) inbound.CreateSubmissionUseCase {
	var attemptRepo outbound.SubmissionAttemptRepository
	if len(attemptRepos) > 0 {
		attemptRepo = attemptRepos[0]
	}
	return &createSubmissionUseCase{
		submissionRepo: submissionRepo,
		attemptRepo:    attemptRepo,
		txManager:      txManager,
		judgePublisher: judgePublisher,
		attemptIDs:     attemptIDs,
		problemReader:  problemReader,
	}
}

func (uc *createSubmissionUseCase) Execute(
	ctx context.Context,
	claims auth.Claims,
	req dto.CreateSubmissionRequest,
) (dto.CreateSubmissionResponse, error) {
	if claims.UserID == "" {
		return dto.CreateSubmissionResponse{}, response.NewAppError(response.CodeUnauthorized, "unauthorized", nil)
	}

	if req.ProblemID <= 0 {
		return dto.CreateSubmissionResponse{}, domain.ErrInvalidProblemID
	}

	language, ok := entity.ParseLanguage(req.Language)
	if !ok || !language.IsExecutable() {
		return dto.CreateSubmissionResponse{}, domain.ErrInvalidLanguage
	}

	if strings.TrimSpace(req.SourceCode) == "" {
		return dto.CreateSubmissionResponse{}, domain.ErrInvalidSourceCode
	}
	if len(req.SourceCode) > entity.MaxSourceCodeBytes {
		return dto.CreateSubmissionResponse{}, domain.ErrSourceCodeTooLarge
	}

	problem, err := uc.problemReader.GetProblem(ctx, req.ProblemID, dto.ProblemActor{
		UserID: claims.UserID,
		Role:   claims.Role,
	})
	if err != nil {
		return dto.CreateSubmissionResponse{}, err
	}

	attemptID := strings.TrimSpace(uc.attemptIDs.NewAttemptID())
	if attemptID == "" {
		return dto.CreateSubmissionResponse{}, domain.ErrInternalServer.Wrap(domain.ErrInvalidJudgeResult)
	}

	submission := entity.NewSubmission(
		problem.ID,
		problem.Title,
		claims.UserID,
		claims.Username,
		language,
		req.SourceCode,
		attemptID,
	)

	err = uc.txManager.ExecuteInTx(ctx, func(txCtx context.Context) error {
		if err := uc.submissionRepo.Create(txCtx, submission); err != nil {
			return err
		}
		if uc.attemptRepo != nil {
			if err := uc.attemptRepo.Create(txCtx, entity.NewSubmissionAttempt(
				submission.ID,
				submission.CurrentAttemptID,
				entity.AttemptTriggerSubmission,
				claims.UserID,
			)); err != nil {
				return err
			}
		}

		job := pkgjudge.JobMessage{
			SubmissionID: submission.ID,
			ProblemID:    submission.ProblemID,
			ProblemSlug:  problem.Slug,
			UserID:       submission.UserID,
			Language:     string(submission.Language),
			SourceCode:   submission.SourceCode,
			AttemptID:    submission.CurrentAttemptID,
			EnqueuedAt:   time.Now().UTC(),
		}

		if err := uc.judgePublisher.Publish(txCtx, job); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return dto.CreateSubmissionResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	return dto.CreateSubmissionResponse{
		ID:           submission.ID,
		ProblemID:    submission.ProblemID,
		ProblemTitle: submission.ProblemName,
		Language:     string(submission.Language),
		Status:       string(submission.Status),
		CreatedAt:    submission.CreatedAt,
	}, nil
}
