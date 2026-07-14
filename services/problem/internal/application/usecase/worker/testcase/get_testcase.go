package worker

import (
	"context"
	"go-judge-system/services/problem/internal/application/dto"
	"go-judge-system/services/problem/internal/application/port/inbound/worker"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"time"
)

const presignedURLexpiry = 15 * time.Minute

type getTestCaseUseCase struct {
	tcRepo    outbound.TestCaseRepository
	tcStorage outbound.TestCaseStorage
}

func NewGetTestCaseUseCase(tcRepo outbound.TestCaseRepository, tcStorage outbound.TestCaseStorage) worker.GetTestCaseUseCase {
	return &getTestCaseUseCase{
		tcRepo:    tcRepo,
		tcStorage: tcStorage,
	}
}

func (uc *getTestCaseUseCase) Execute(ctx context.Context, problemID int64) (*dto.InternalTestCaseResponse, error) {
	testcase, err := uc.tcRepo.GetByProblemID(ctx, problemID)
	if err != nil {
		return nil, err
	}

	downloadURL, err := uc.tcStorage.GetPresignedURL(ctx, testcase.ZipObjectKey, presignedURLexpiry)
	if err != nil {
		return nil, err
	}

	return &dto.InternalTestCaseResponse{
		TestCount:      testcase.TestCount,
		Version:        testcase.Version,
		ZipDownloadURL: downloadURL,
	}, nil
}
