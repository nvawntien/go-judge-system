package testcase

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/problem/internal/application/dto"
	"go-judge-system/services/problem/internal/application/port/inbound"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"
	"go-judge-system/services/problem/internal/domain/entity"

	"go.uber.org/zap"
)

const (
	maxZipFileSize         = 100 * 1024 * 1024 // 100 MB compressed
	maxUncompressedSize    = 500 * 1024 * 1024 // 500 MB total uncompressed (zip-bomb guard)
	maxFileCountInZip      = 2000              // 1000 test cases max (1000 .in + 1000 .out)
)

// testcaseFilePattern matches valid testcase file names: "1.in", "1.out", "42.in", etc.
var testcaseFilePattern = regexp.MustCompile(`^(\d+)\.(in|out)$`)

type uploadTestCaseUseCase struct {
	problemRepo outbound.ProblemRepository
	tcRepo      outbound.TestCaseRepository
	storage     outbound.ObjectStorage
	logger      *zap.Logger
}

func NewUploadTestCaseUseCase(
	problemRepo outbound.ProblemRepository,
	tcRepo outbound.TestCaseRepository,
	storage outbound.ObjectStorage,
	logger *zap.Logger,
) inbound.UploadTestCaseUseCase {
	return &uploadTestCaseUseCase{
		problemRepo: problemRepo,
		tcRepo:      tcRepo,
		storage:     storage,
		logger:      logger,
	}
}

func (uc *uploadTestCaseUseCase) Execute(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest, req dto.UploadTestCaseRequest) (dto.UploadTestCasesResponse, error) {
	problem, err := uc.problemRepo.GetByID(ctx, params.ID)
	if err != nil {
		if !errors.Is(err, domain.ErrProblemNotFound) {
			uc.logger.Error("failed to get problem", zap.Error(err))
			return dto.UploadTestCasesResponse{}, domain.ErrInternalServer.Wrap(err)
		}
		return dto.UploadTestCasesResponse{}, domain.ErrProblemNotFound
	}

	if !claims.CanManage(problem.AuthorID) {
		return dto.UploadTestCasesResponse{}, domain.ErrNotOwner
	}

	if req.File.Size > maxZipFileSize {
		return dto.UploadTestCasesResponse{}, domain.ErrInvalidTestCase.Wrap(
			fmt.Errorf("zip file too large: %d bytes (max %d)", req.File.Size, maxZipFileSize),
		)
	}

	src, err := req.File.Open()
	if err != nil {
		uc.logger.Error("failed to open multipart file", zap.Error(err))
		return dto.UploadTestCasesResponse{}, domain.ErrInternalServer.Wrap(err)
	}
	defer src.Close()

	tmpFile, err := os.CreateTemp("", "tc-upload-*.zip")
	if err != nil {
		uc.logger.Error("failed to create temp file", zap.Error(err))
		return dto.UploadTestCasesResponse{}, domain.ErrInternalServer.Wrap(err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, src); err != nil {
		tmpFile.Close()
		uc.logger.Error("failed to stream upload to temp file", zap.Error(err))
		return dto.UploadTestCasesResponse{}, domain.ErrInternalServer.Wrap(err)
	}
	tmpFile.Close()

	testCount, err := validateTestCaseZip(tmpPath)
	if err != nil {
		uc.logger.Warn("zip validation failed",
			zap.Int64("problem_id", params.ID),
			zap.Error(err),
		)
		return dto.UploadTestCasesResponse{}, domain.ErrInvalidTestCase.Wrap(err)
	}

	version := strconv.FormatInt(time.Now().Unix(), 10)
	objectKey := fmt.Sprintf("problems/%d/testcases_%s.zip", problem.ID, version)

	if err := uc.storage.UploadFromFile(ctx, objectKey, tmpPath); err != nil {
		uc.logger.Error("failed to upload zip to object storage",
			zap.Int64("problem_id", problem.ID),
			zap.String("object_key", objectKey),
			zap.Error(err),
		)
		return dto.UploadTestCasesResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	tc := entity.NewTestCase(problem.ID, objectKey, testCount, version)

	if err := uc.tcRepo.Upsert(ctx, tc); err != nil {
		uc.logger.Error("failed to upsert testcase metadata",
			zap.Int64("problem_id", problem.ID),
			zap.Error(err),
		)

		if delErr := uc.storage.DeleteObject(ctx, objectKey); delErr != nil {
			uc.logger.Error("failed to rollback orphan object", zap.Error(delErr))
		}
		return dto.UploadTestCasesResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	uc.logger.Info("testcases uploaded successfully",
		zap.Int64("problem_id", problem.ID),
		zap.Int("test_count", testCount),
		zap.String("version", version),
	)

	return dto.UploadTestCasesResponse{
		TestCount: testCount,
		Version:   version,
	}, nil
}

// validateTestCaseZip opens a ZIP file on disk and validates its structure.
// It checks for:
//   - valid zip format
//   - zip bomb (total uncompressed size)
//   - max file count
//   - naming convention: {N}.in / {N}.out (N starts from 1, consecutive)
//   - every .in has a matching .out and vice-versa
//   - no subdirectories or unexpected files
//
// Returns the number of test-case pairs on success.
func validateTestCaseZip(filePath string) (int, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return 0, fmt.Errorf("invalid zip file: %w", err)
	}
	defer r.Close()

	if len(r.File) == 0 {
		return 0, fmt.Errorf("zip file is empty")
	}

	if len(r.File) > maxFileCountInZip {
		return 0, fmt.Errorf("too many files in zip: %d (max %d)", len(r.File), maxFileCountInZip)
	}

	// Track uncompressed size for zip-bomb detection
	var totalUncompressed uint64

	// Collect test numbers separately for .in and .out
	inSet := make(map[int]struct{})
	outSet := make(map[int]struct{})

	for _, f := range r.File {
		// Reject directories
		if f.FileInfo().IsDir() {
			return 0, fmt.Errorf("unexpected directory in zip: %q", f.Name)
		}

		// Reject files in subdirectories (path contains separator)
		if strings.ContainsRune(f.Name, '/') || strings.ContainsRune(f.Name, '\\') {
			return 0, fmt.Errorf("files must be at root of zip, found: %q", f.Name)
		}

		// Zip-bomb guard
		totalUncompressed += f.UncompressedSize64
		if totalUncompressed > maxUncompressedSize {
			return 0, fmt.Errorf("total uncompressed size exceeds limit (%d bytes)", maxUncompressedSize)
		}

		// Match naming pattern
		matches := testcaseFilePattern.FindStringSubmatch(f.Name)
		if matches == nil {
			return 0, fmt.Errorf("invalid file name %q: expected format {N}.in or {N}.out", f.Name)
		}

		num, err := strconv.Atoi(matches[1])
		if err != nil || num <= 0 {
			return 0, fmt.Errorf("invalid test number in %q: must be a positive integer", f.Name)
		}

		ext := matches[2]
		switch ext {
		case "in":
			if _, dup := inSet[num]; dup {
				return 0, fmt.Errorf("duplicate input file: %q", f.Name)
			}
			inSet[num] = struct{}{}
		case "out":
			if _, dup := outSet[num]; dup {
				return 0, fmt.Errorf("duplicate output file: %q", f.Name)
			}
			outSet[num] = struct{}{}
		}
	}

	testCount := len(inSet)

	// Every .in must have a matching .out and vice-versa
	if testCount != len(outSet) {
		return 0, fmt.Errorf("mismatched in/out count: %d .in files vs %d .out files", testCount, len(outSet))
	}

	for num := range inSet {
		if _, ok := outSet[num]; !ok {
			return 0, fmt.Errorf("missing output file for test #%d (have %d.in but no %d.out)", num, num, num)
		}
	}

	// Numbers must be consecutive starting from 1
	nums := make([]int, 0, testCount)
	for n := range inSet {
		nums = append(nums, n)
	}
	sort.Ints(nums)

	for i, n := range nums {
		expected := i + 1
		if n != expected {
			return 0, fmt.Errorf("test case numbers must be consecutive from 1: expected %d, got %d", expected, n)
		}
	}

	return testCount, nil
}
