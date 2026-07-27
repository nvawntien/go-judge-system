package entity

import "time"

type Language string

const (
	LanguageC          Language = "C"
	LanguageCPP        Language = "CPP"
	LanguageJava       Language = "JAVA"
	LanguagePython     Language = "PYTHON"
	LanguageGo         Language = "GO"
	LanguageJavaScript Language = "JAVASCRIPT"
	MaxSourceCodeBytes          = 256 * 1024
)

func (l Language) IsExecutable() bool {
	switch l {
	case LanguageCPP, LanguageGo, LanguagePython, LanguageJava:
		return true
	default:
		return false
	}
}

type Status string

const (
	StatusPending           Status = "PENDING"
	StatusJudging           Status = "JUDGING"
	StatusAccepted          Status = "ACCEPTED"
	StatusWrongAnswer       Status = "WRONG_ANSWER"
	StatusTimeLimitExceed   Status = "TIME_LIMIT_EXCEEDED"
	StatusMemoryLimitExceed Status = "MEMORY_LIMIT_EXCEEDED"
	StatusRuntimeError      Status = "RUNTIME_ERROR"
	StatusCompilationError  Status = "COMPILATION_ERROR"
	StatusSystemError       Status = "SYSTEM_ERROR"
)

func ParseStatus(value string) (Status, bool) {
	switch Status(value) {
	case StatusPending,
		StatusJudging,
		StatusAccepted,
		StatusWrongAnswer,
		StatusTimeLimitExceed,
		StatusMemoryLimitExceed,
		StatusRuntimeError,
		StatusCompilationError,
		StatusSystemError:
		return Status(value), true
	default:
		return "", false
	}
}

type Submission struct {
	ID               int64
	ProblemID        int64
	ProblemName      string
	UserID           string
	Username         string
	Language         Language
	SourceCode       string
	CurrentAttemptID string
	Status           Status
	ExecutionTime    *int
	MemoryUsed       *int
	CompileOutput    *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewSubmission(problemID int64, problemName, userID, username string, language Language, sourceCode, currentAttemptID string) *Submission {
	now := time.Now()
	return &Submission{
		ProblemID:        problemID,
		ProblemName:      problemName,
		UserID:           userID,
		Username:         username,
		Language:         language,
		SourceCode:       sourceCode,
		CurrentAttemptID: currentAttemptID,
		Status:           StatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func ParseLanguage(value string) (Language, bool) {
	switch Language(value) {
	case LanguageC, LanguageCPP, LanguageJava, LanguagePython, LanguageGo, LanguageJavaScript:
		return Language(value), true
	default:
		return "", false
	}
}

func (s *Submission) MarkJudging() {
	s.Status = StatusJudging
	s.UpdatedAt = time.Now()
}

func (s *Submission) MarkCompleted(status Status, timeUsed, memoryUsed *int, compileOutput *string) {
	s.Status = status
	s.ExecutionTime = timeUsed
	s.MemoryUsed = memoryUsed
	s.CompileOutput = compileOutput
	s.UpdatedAt = time.Now()
}

func (s *Submission) ResetForRejudge() {
	s.Status = StatusPending
	s.ExecutionTime = nil
	s.MemoryUsed = nil
	s.CompileOutput = nil
	s.UpdatedAt = time.Now()
}
