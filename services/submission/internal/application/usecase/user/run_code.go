package user

import (
	"context"
	"strings"
	"time"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/response"
	"go-judge-system/services/submission/internal/application/dto"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/user"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"
)

const (
	runKindSample = "sample"
	runKindCustom = "custom"
)

type runCodeUseCase struct {
	problemReader outbound.ProblemReader
	judgeRunner   outbound.JudgeRunner
	limits        dto.RunCodeLimits
}

func NewRunCodeUseCase(
	problemReader outbound.ProblemReader,
	judgeRunner outbound.JudgeRunner,
	limits dto.RunCodeLimits,
) inbound.RunCodeUseCase {
	return &runCodeUseCase{
		problemReader: problemReader,
		judgeRunner:   judgeRunner,
		limits:        normalizeRunCodeLimits(limits),
	}
}

func (uc *runCodeUseCase) Execute(
	ctx context.Context,
	claims auth.Claims,
	req dto.RunCodeRequest,
) (dto.RunCodeResponse, error) {
	if claims.UserID == "" {
		return dto.RunCodeResponse{}, response.NewAppError(response.CodeUnauthorized, "unauthorized", nil)
	}
	if req.ProblemID <= 0 {
		return dto.RunCodeResponse{}, domain.ErrInvalidProblemID
	}

	language, ok := entity.ParseLanguage(req.Language)
	if !ok || !language.IsExecutable() {
		return dto.RunCodeResponse{}, domain.ErrUnsupportedRunLanguage
	}
	if strings.TrimSpace(req.SourceCode) == "" {
		return dto.RunCodeResponse{}, domain.ErrInvalidSourceCode
	}
	if len(req.SourceCode) > uc.limits.MaxSourceCodeBytes {
		return dto.RunCodeResponse{}, domain.ErrRunPayloadTooLarge
	}
	if len(req.TestCases) == 0 || len(req.TestCases) > uc.limits.MaxTestCases {
		return dto.RunCodeResponse{}, domain.ErrInvalidRunTestCase
	}

	testCases, err := uc.validateTestCases(req.TestCases)
	if err != nil {
		return dto.RunCodeResponse{}, err
	}

	problem, err := uc.problemReader.GetProblem(ctx, req.ProblemID, dto.ProblemActor{
		UserID: claims.UserID,
		Role:   claims.Role,
	})
	if err != nil {
		return dto.RunCodeResponse{}, err
	}

	callCtx, cancel := context.WithTimeout(ctx, uc.limits.RequestTimeout)
	defer cancel()

	return uc.judgeRunner.RunCode(callCtx, outbound.JudgeRunRequest{
		Language:         string(language),
		SourceCode:       req.SourceCode,
		TestCases:        testCases,
		TimeLimitMS:      problemTimeLimitMS(problem.TimeLimit, uc.limits.DefaultTimeLimit),
		MemoryLimitKB:    problemMemoryLimitKB(problem.MemoryLimit, uc.limits.DefaultMemoryLimitKB),
		OutputLimitBytes: uc.limits.DefaultOutputLimit,
	})
}

func (uc *runCodeUseCase) validateTestCases(inputs []dto.RunTestCaseInput) ([]outbound.JudgeRunTestCase, error) {
	seen := make(map[string]struct{}, len(inputs))
	testCases := make([]outbound.JudgeRunTestCase, 0, len(inputs))

	for _, input := range inputs {
		id := strings.TrimSpace(input.ID)
		kind := strings.TrimSpace(input.Kind)
		if id == "" {
			return nil, domain.ErrInvalidRunTestCase
		}
		if _, exists := seen[id]; exists {
			return nil, domain.ErrInvalidRunTestCase.Wrap(response.NewAppError(response.CodeBadRequest, "duplicate testcase id", nil))
		}
		seen[id] = struct{}{}

		switch kind {
		case runKindSample:
			if input.ExpectedOutput == nil {
				return nil, domain.ErrInvalidRunTestCase.Wrap(response.NewAppError(response.CodeBadRequest, "sample testcase requires expected_output", nil))
			}
		case runKindCustom:
		default:
			return nil, domain.ErrInvalidRunTestCase
		}

		if len(input.Stdin) > uc.limits.MaxStdinBytes {
			return nil, domain.ErrRunPayloadTooLarge
		}
		if input.ExpectedOutput != nil && len(*input.ExpectedOutput) > uc.limits.MaxExpectedOutputBytes {
			return nil, domain.ErrRunPayloadTooLarge
		}

		testCases = append(testCases, outbound.JudgeRunTestCase{
			ID:             id,
			Kind:           kind,
			Stdin:          input.Stdin,
			ExpectedOutput: input.ExpectedOutput,
		})
	}

	return testCases, nil
}

func normalizeRunCodeLimits(limits dto.RunCodeLimits) dto.RunCodeLimits {
	if limits.MaxTestCases <= 0 {
		limits.MaxTestCases = 10
	}
	if limits.MaxSourceCodeBytes <= 0 {
		limits.MaxSourceCodeBytes = entity.MaxSourceCodeBytes
	}
	if limits.MaxStdinBytes <= 0 {
		limits.MaxStdinBytes = 64 * 1024
	}
	if limits.MaxExpectedOutputBytes <= 0 {
		limits.MaxExpectedOutputBytes = 64 * 1024
	}
	if limits.MaxCapturedOutputBytes <= 0 {
		limits.MaxCapturedOutputBytes = 1024 * 1024
	}
	if limits.RequestTimeout <= 0 {
		limits.RequestTimeout = 30 * time.Second
	}
	if limits.DefaultTimeLimit <= 0 {
		limits.DefaultTimeLimit = 2 * time.Second
	}
	if limits.DefaultMemoryLimitKB <= 0 {
		limits.DefaultMemoryLimitKB = 256 * 1024
	}
	if limits.DefaultOutputLimit <= 0 {
		limits.DefaultOutputLimit = int64(limits.MaxCapturedOutputBytes)
	}
	return limits
}

func problemTimeLimitMS(value float64, fallback time.Duration) int64 {
	if value <= 0 {
		return int64(fallback / time.Millisecond)
	}
	if value >= 50 {
		return int64(value)
	}
	return int64(value * 1000)
}

func problemMemoryLimitKB(value int, fallback int64) int64 {
	if value <= 0 {
		return fallback
	}
	return int64(value) * 1024
}
