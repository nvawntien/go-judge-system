package result

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	pkgjudge "go-judge-system/pkg/judge"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/result"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"

	"go.uber.org/zap"
)

const (
	maxStoredErrorMessageBytes = 4096
	publicSystemErrorMessage   = "The judge could not complete this submission."
	publicTimeLimitMessage     = "Execution exceeded the time limit."
	publicMemoryLimitMessage   = "Execution exceeded the memory limit."
	publicOutputLimitMessage   = "Execution exceeded the output limit."
)

var (
	errorPathWithSource = regexp.MustCompile(`(?:/tmp/|/w/|/workspace/|/app/workspace/|/judge/)[^:\s"]*/(main\.go|main\.cpp|main\.py|Main\.java)`)
	errorInternalPrefix = regexp.MustCompile(`(?:/tmp/|/w/|/workspace/|/app/workspace/|/judge/)+`)
	errorRelativeSource = regexp.MustCompile(`\./(main\.go|main\.cpp|main\.py|Main\.java)`)
	datasetSHA256Hex    = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type applyJudgeResultUseCase struct {
	submissionRepo outbound.SubmissionRepository
	resultRepo     outbound.SubmissionResultRepository
	attemptRepo    outbound.SubmissionAttemptRepository
	txManager      outbound.TransactionManager
	eventHub       outbound.SubmissionEventHub
	logger         *zap.Logger
}

func NewApplyJudgeResultUseCase(
	submissionRepo outbound.SubmissionRepository,
	resultRepo outbound.SubmissionResultRepository,
	txManager outbound.TransactionManager,
	eventHub outbound.SubmissionEventHub,
	logger *zap.Logger,
	attemptRepos ...outbound.SubmissionAttemptRepository,
) inbound.ApplyJudgeResultUseCase {
	var attemptRepo outbound.SubmissionAttemptRepository
	if len(attemptRepos) > 0 {
		attemptRepo = attemptRepos[0]
	}
	return &applyJudgeResultUseCase{
		submissionRepo: submissionRepo,
		resultRepo:     resultRepo,
		attemptRepo:    attemptRepo,
		txManager:      txManager,
		eventHub:       eventHub,
		logger:         logger,
	}
}

func (uc *applyJudgeResultUseCase) Execute(ctx context.Context, msg pkgjudge.ResultMessage) error {
	if msg.SubmissionID <= 0 || strings.TrimSpace(msg.AttemptID) == "" {
		return domain.ErrInvalidJudgeResult
	}

	status, err := mapTerminalSubmissionStatus(msg.Status)
	if err != nil {
		return err
	}
	if err := validateDatasetChecksum(msg.DatasetChecksum); err != nil {
		return err
	}

	results, err := mapTestCaseResults(msg.SubmissionID, msg.AttemptID, msg.TestCases)
	if err != nil {
		return err
	}

	var eventToPublish *entity.SubmissionEvent
	err = uc.txManager.ExecuteInTx(ctx, func(txCtx context.Context) error {
		submission, err := uc.submissionRepo.GetByIDForUpdate(txCtx, msg.SubmissionID)
		if err != nil {
			return err
		}

		if strings.TrimSpace(submission.CurrentAttemptID) == "" {
			return nil
		}
		if submission.CurrentAttemptID != msg.AttemptID {
			return nil
		}
		if entity.IsTerminalStatus(submission.Status) {
			return nil
		}

		compileOutput, errorMessage := judgeOutputFields(status, msg)
		submission.MarkCompleted(status, msg.ExecutionTime, msg.MemoryUsed, compileOutput, errorMessage)
		if err := uc.submissionRepo.Update(txCtx, submission); err != nil {
			return err
		}

		if err := uc.resultRepo.ReplaceBySubmissionIDAndAttemptID(txCtx, msg.SubmissionID, msg.AttemptID, results); err != nil {
			return err
		}
		if uc.attemptRepo != nil {
			if err := uc.attemptRepo.MarkCompleted(
				txCtx,
				msg.AttemptID,
				status,
				msg.TestcaseVersion,
				msg.TestCount,
				msg.DatasetChecksum,
			); err != nil {
				return err
			}
		}

		event := submission.Event()
		eventToPublish = &event
		return nil
	})
	if err != nil {
		return err
	}

	if eventToPublish != nil && uc.eventHub != nil {
		uc.eventHub.Publish(*eventToPublish)
		if uc.logger != nil {
			uc.logger.Debug(
				"published submission event after judge result commit",
				zap.Int64("submission_id", eventToPublish.SubmissionID),
				zap.String("attempt_id", eventToPublish.AttemptID),
				zap.String("status", eventToPublish.Status),
			)
		}
	}
	return nil
}

func validateDatasetChecksum(value *string) error {
	if value == nil {
		return nil
	}
	if !datasetSHA256Hex.MatchString(*value) {
		return domain.ErrInvalidJudgeResult
	}
	return nil
}

func judgeOutputFields(status entity.Status, msg pkgjudge.ResultMessage) (*string, *string) {
	switch status {
	case entity.StatusCompilationError:
		return nonBlankString(msg.CompileOutput), nil
	case entity.StatusRuntimeError:
		return nil, publicErrorMessage(msg.ErrorMessage)
	case entity.StatusTimeLimitExceed:
		if message := publicErrorMessage(msg.ErrorMessage); message != nil {
			return nil, message
		}
		message := publicTimeLimitMessage
		return nil, &message
	case entity.StatusMemoryLimitExceed:
		if message := publicErrorMessage(msg.ErrorMessage); message != nil {
			return nil, message
		}
		message := publicMemoryLimitMessage
		return nil, &message
	case entity.StatusOutputLimitExceed:
		if message := publicErrorMessage(msg.ErrorMessage); message != nil {
			return nil, message
		}
		message := publicOutputLimitMessage
		return nil, &message
	case entity.StatusSystemError:
		message := publicSystemErrorMessage
		return nil, &message
	default:
		return nil, nil
	}
}

func publicErrorMessage(value *string) *string {
	if value == nil {
		return nil
	}
	message := errorPathWithSource.ReplaceAllString(*value, "$1")
	message = errorRelativeSource.ReplaceAllString(message, "$1")
	message = errorInternalPrefix.ReplaceAllString(message, "")
	message = strings.ToValidUTF8(message, "")
	message = stripControlCharacters(message)
	message = strings.TrimSpace(message)
	if message == "" {
		return nil
	}
	if len(message) > maxStoredErrorMessageBytes {
		message = message[:maxStoredErrorMessageBytes]
		for !utf8.ValidString(message) && len(message) > 0 {
			message = message[:len(message)-1]
		}
		message = strings.TrimSpace(message) + "…"
	}
	return &message
}

func stripControlCharacters(value string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return r
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, value)
}

func nonBlankString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return value
}

func mapTerminalSubmissionStatus(raw string) (entity.Status, error) {
	status, ok := entity.ParseStatus(raw)
	if !ok {
		return "", domain.ErrInvalidSubmissionStatus
	}

	switch status {
	case entity.StatusAccepted,
		entity.StatusWrongAnswer,
		entity.StatusTimeLimitExceed,
		entity.StatusMemoryLimitExceed,
		entity.StatusOutputLimitExceed,
		entity.StatusRuntimeError,
		entity.StatusCompilationError,
		entity.StatusSystemError:
		return status, nil
	default:
		return "", domain.ErrInvalidSubmissionStatus
	}
}

func mapTestCaseResults(submissionID int64, attemptID string, items []pkgjudge.TestCaseResultItem) ([]*entity.SubmissionResult, error) {
	results := make([]*entity.SubmissionResult, 0, len(items))
	for _, item := range items {
		if item.Index <= 0 {
			return nil, domain.ErrInvalidJudgeResult
		}

		status, err := mapTestCaseStatus(item.Status)
		if err != nil {
			return nil, err
		}

		results = append(results, &entity.SubmissionResult{
			SubmissionID:   submissionID,
			AttemptID:      attemptID,
			TestIndex:      item.Index,
			Status:         status,
			ActualOutput:   item.ActualOutput,
			Input:          item.Input,
			ExpectedOutput: item.ExpectedOutput,
			ExecutionTime:  item.ExecutionTime,
			MemoryUsed:     item.MemoryUsed,
		})
	}

	return results, nil
}

func mapTestCaseStatus(raw string) (entity.ResultStatus, error) {
	switch entity.ResultStatus(raw) {
	case entity.ResultAccepted,
		entity.ResultWrongAnswer,
		entity.ResultTimeLimit,
		entity.ResultMemoryLimit,
		entity.ResultOutputLimit,
		entity.ResultRuntimeError,
		entity.ResultSystemError:
		return entity.ResultStatus(raw), nil
	default:
		return "", domain.ErrInvalidJudgeResult
	}
}
