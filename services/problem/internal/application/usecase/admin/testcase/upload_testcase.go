package testcase

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/problem/internal/application/dto"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"
	"go-judge-system/services/problem/internal/domain/entity"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	maxFileZipSize   = 50 << 20  // 50MB
	maxExtractedSize = 100 << 20 // 100MB
	maxFileCount     = 2000      // 1000 .in and .out files
)

// This regex pattern is used to validate the file names of the test cases. The expected format is "<number>.<extension>", where <number> is a sequence of digits and <extension> is either "in" or "out". For example, valid file names include "1.in", "2.out", "123.in", etc.
var testcaseFilePattern = regexp.MustCompile(`^(\d+)\.(in|out)$`)

type uploadTestCaseUseCase struct {
	problemRepo     outbound.ProblemRepository
	testcaseRepo    outbound.TestCaseRepository
	testcaseStorage outbound.TestCaseStorage
}

func NewUploadTestCaseUseCase(problemRepo outbound.ProblemRepository, testcaseRepo outbound.TestCaseRepository, testcaseStorage outbound.TestCaseStorage) inbound.UploadTestCaseUseCase {
	return &uploadTestCaseUseCase{
		problemRepo:     problemRepo,
		testcaseRepo:    testcaseRepo,
		testcaseStorage: testcaseStorage,
	}
}

func (uc *uploadTestCaseUseCase) Execute(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest, req dto.UploadTestCaseRequest) (dto.TestCaseMetadataResponse, error) {
	if !claims.Role.AtLeast(rbac.RoleContributor) {
		return dto.TestCaseMetadataResponse{}, domain.ErrForbidden
	}

	problem, err := uc.problemRepo.GetByID(ctx, params.ID)
	if err != nil {
		if errors.Is(err, domain.ErrProblemNotFound) {
			return dto.TestCaseMetadataResponse{}, domain.ErrProblemNotFound
		}
		return dto.TestCaseMetadataResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	if !claims.Role.AtLeast(rbac.RoleModerator) {
		if problem.AuthorID != claims.UserID {
			return dto.TestCaseMetadataResponse{}, domain.ErrForbidden
		}

		if !problem.IsHidden {
			return dto.TestCaseMetadataResponse{}, domain.ErrForbidden
		}
	}

	if req.File == nil {
		return dto.TestCaseMetadataResponse{}, domain.ErrInvalidTestCase.Wrap(fmt.Errorf("testcase file is required"))
	}

	if req.File.Size > maxFileZipSize {
		return dto.TestCaseMetadataResponse{}, domain.ErrInvalidTestCase.Wrap(fmt.Errorf("testcase file size exceeds the limit of %d bytes", maxFileZipSize))
	}

	src, err := req.File.Open()
	if err != nil {
		return dto.TestCaseMetadataResponse{}, domain.ErrInternalServer.Wrap(err)
	}
	defer src.Close()

	tmpFile, err := os.CreateTemp("", "testcase-upload-*.zip")
	if err != nil {
		return dto.TestCaseMetadataResponse{}, domain.ErrInternalServer.Wrap(err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, src); err != nil {
		_ = tmpFile.Close()
		return dto.TestCaseMetadataResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	if err := tmpFile.Close(); err != nil {
		return dto.TestCaseMetadataResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	testCount, err := validateTestCaseZip(tmpPath)
	if err != nil {
		return dto.TestCaseMetadataResponse{}, domain.ErrInvalidTestCase.Wrap(err)
	}

	version := 1
	var oldObjectKey string

	oldTC, err := uc.testcaseRepo.GetByProblemID(ctx, problem.ID)
	if err == nil {
		version = oldTC.Version + 1
		oldObjectKey = oldTC.ZipObjectKey
	} else if !errors.Is(err, domain.ErrTestCaseNotFound) {
		return dto.TestCaseMetadataResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	objectKey := fmt.Sprintf("problems/%d/testcases/v%d.zip", problem.ID, version)

	if err := uc.testcaseStorage.UploadTestCase(ctx, objectKey, tmpPath); err != nil {
		return dto.TestCaseMetadataResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	tc := entity.NewTestCase(problem.ID, objectKey, testCount, version)

	if err := uc.testcaseRepo.Upsert(ctx, tc); err != nil {
		_ = uc.testcaseStorage.DeleteTestCase(ctx, objectKey)
		return dto.TestCaseMetadataResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	if oldObjectKey != "" {
		_ = uc.testcaseStorage.DeleteTestCase(ctx, oldObjectKey)
	}

	return dto.TestCaseMetadataResponse{
		ProblemID:    tc.ProblemID,
		ZipObjectKey: tc.ZipObjectKey,
		TestCount:    tc.TestCount,
		Version:      tc.Version,
		CreatedAt:    tc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    tc.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func validateTestCaseZip(filePath string) (int, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return 0, fmt.Errorf("invalid zip file: %w", err)
	}
	defer r.Close()

	if len(r.File) == 0 {
		return 0, fmt.Errorf("zip file is empty")
	}

	if len(r.File) > maxFileCount {
		return 0, fmt.Errorf("too many files in zip: %d, max %d", len(r.File), maxFileCount)
	}

	var totalUncompressed uint64

	inSet := make(map[int]struct{})
	outSet := make(map[int]struct{})

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			return 0, fmt.Errorf("unexpected directory in zip: %q", f.Name)
		}

		if strings.ContainsRune(f.Name, '/') || strings.ContainsRune(f.Name, '\\') {
			return 0, fmt.Errorf("files must be at root of zip, found: %q", f.Name)
		}

		totalUncompressed += f.UncompressedSize64
		if totalUncompressed > maxExtractedSize {
			return 0, fmt.Errorf("total uncompressed size exceeds limit: %d bytes", maxExtractedSize)
		}

		matches := testcaseFilePattern.FindStringSubmatch(f.Name)
		if matches == nil {
			return 0, fmt.Errorf("invalid file name %q: expected {N}.in or {N}.out", f.Name)
		}

		num, err := strconv.Atoi(matches[1])
		if err != nil || num <= 0 {
			return 0, fmt.Errorf("invalid test number in %q: must be a positive integer", f.Name)
		}

		switch matches[2] {
		case "in":
			if _, exists := inSet[num]; exists {
				return 0, fmt.Errorf("duplicate input file: %q", f.Name)
			}

			inSet[num] = struct{}{}

		case "out":
			if _, exists := outSet[num]; exists {
				return 0, fmt.Errorf("duplicate output file: %q", f.Name)
			}

			outSet[num] = struct{}{}
		}
	}

	testCount := len(inSet)
	if testCount == 0 {
		return 0, fmt.Errorf("no testcase pairs found")
	}

	if testCount != len(outSet) {
		return 0, fmt.Errorf("mismatched in/out count: %d .in files vs %d .out files", testCount, len(outSet))
	}

	for num := range inSet {
		if _, ok := outSet[num]; !ok {
			return 0, fmt.Errorf("missing output file for test #%d", num)
		}
	}

	nums := make([]int, 0, testCount)
	for n := range inSet {
		nums = append(nums, n)
	}

	sort.Ints(nums)

	for i, n := range nums {
		expected := i + 1
		if n != expected {
			return 0, fmt.Errorf("testcase numbers must be consecutive from 1: expected %d, got %d", expected, n)
		}
	}

	return testCount, nil
}
