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
)

type Status string

const (
	StatusPending           Status = "PENDING"
	StatusJudging           Status = "JUDGING"
	StatusAccepted          Status = "ACCEPTED"
	StatusWrongAnswer       Status = "WRONG_ANSWER"
	StatusTimeLimitExceed   Status = "TLE"
	StatusMemoryLimitExceed Status = "MLE"
	StatusRuntimeError      Status = "RUNTIME_ERROR"
	StatusCompilationError  Status = "COMPILATION_ERROR"
)

type Submission struct {
	ID            int64
	ProblemID     int64
	ProblemName   string
	UserID        string
	Username      string
	Language      Language
	SourceCode    string
	Status        Status
	ExecutionTime *int
	MemoryUsed    *int
	CompileOutput *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewSubmission(problemID int64, problemName, userID, username string, language Language, sourceCode string) *Submission {
	now := time.Now()
	return &Submission{
		ProblemID:   problemID,
		ProblemName: problemName,
		UserID:      userID,
		Username:    username,
		Language:    language,
		SourceCode:  sourceCode,
		Status:      StatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
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
