package testcase

import (
	"context"
	"errors"
	"time"

	"go-judge-system/services/problem/internal/application/dto"
	"go-judge-system/services/problem/internal/application/port/inbound"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"

	"go.uber.org/zap"
)

const presignedURLExpiry = 1 * time.Hour

type getTestCaseForWorkerUseCase struct {
	tcRepo  outbound.TestCaseRepository
	storage outbound.ObjectStorage
	logger  *zap.Logger
}

func NewGetTestCaseForWorkerUseCase(
	tcRepo outbound.TestCaseRepository,
	storage outbound.ObjectStorage,
	logger *zap.Logger,
) inbound.GetTestCaseForWorkerUseCase {
	return &getTestCaseForWorkerUseCase{tcRepo: tcRepo, storage: storage, logger: logger}
}

func (uc *getTestCaseForWorkerUseCase) Execute(ctx context.Context, params dto.ProblemIDRequest) (dto.InternalTestCaseResponse, error) {
	tc, err := uc.tcRepo.GetByProblemID(ctx, params.ID)
	if err != nil {
		if errors.Is(err, domain.ErrTestCaseNotFound) {
			return dto.InternalTestCaseResponse{}, domain.ErrTestCaseNotFound
		}
		uc.logger.Error("failed to get testcase metadata",
			zap.Int64("problem_id", params.ID),
			zap.Error(err),
		)
		return dto.InternalTestCaseResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	downloadURL, err := uc.storage.GetPresignedURL(ctx, tc.ZipObjectKey, presignedURLExpiry)
	if err != nil {
		uc.logger.Error("failed to generate presigned URL",
			zap.Int64("problem_id", params.ID),
			zap.String("object_key", tc.ZipObjectKey),
			zap.Error(err),
		)
		return dto.InternalTestCaseResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	return dto.InternalTestCaseResponse{
		ProblemID:      tc.ProblemID,
		TestCount:      tc.TestCount,
		Version:        tc.Version,
		ZipDownloadURL: downloadURL,
	}, nil
}
