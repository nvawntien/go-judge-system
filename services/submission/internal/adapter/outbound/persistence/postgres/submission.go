package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SubmissionDAO struct {
	ID               int64     `gorm:"primaryKey;autoIncrement"`
	ProblemID        int64     `gorm:"not null;index"`
	ProblemName      string    `gorm:"not null;size:500"`
	UserID           string    `gorm:"not null;size:100;index;index:idx_submission_user_created,priority:1"`
	Username         string    `gorm:"not null;size:255"`
	Language         string    `gorm:"type:varchar(20);not null;index"`
	SourceCode       string    `gorm:"type:text;not null"`
	CurrentAttemptID string    `gorm:"column:current_attempt_id;type:varchar(64);index"`
	Status           string    `gorm:"type:varchar(30);not null;index"`
	ExecutionTime    *int      `gorm:"type:int"`
	MemoryUsed       *int      `gorm:"type:int"`
	CompileOutput    *string   `gorm:"type:text"`
	ErrorMessage     *string   `gorm:"type:text"`
	CreatedAt        time.Time `gorm:"autoCreateTime;index;index:idx_submission_user_created,priority:2"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime"`
}

func (SubmissionDAO) TableName() string { return "submissions" }

type submissionRepository struct {
	db *gorm.DB
}

func NewSubmissionRepository(db *gorm.DB) outbound.SubmissionRepository {
	db.AutoMigrate(&SubmissionDAO{})
	return &submissionRepository{db: db}
}

func NewProfileStatsRepository(db *gorm.DB) outbound.ProfileStatsRepository {
	return &submissionRepository{db: db}
}

func NewSubmissionStreamSnapshotRepository(db *gorm.DB) outbound.SubmissionStreamSnapshotRepository {
	return &submissionRepository{db: db}
}

func (r *submissionRepository) Create(ctx context.Context, submission *entity.Submission) error {
	dao := toSubmissionDAO(submission)
	db := getDB(ctx, r.db)
	if err := db.Create(dao).Error; err != nil {
		return fmt.Errorf("create submission: %w", err)
	}

	submission.ID = dao.ID
	return nil
}

func (r *submissionRepository) GetByID(ctx context.Context, id int64) (*entity.Submission, error) {
	return r.getByID(ctx, id, false)
}

func (r *submissionRepository) GetByIDForUpdate(ctx context.Context, id int64) (*entity.Submission, error) {
	return r.getByID(ctx, id, true)
}

func (r *submissionRepository) GetStreamSnapshot(
	ctx context.Context,
	submissionID int64,
) (*entity.SubmissionStreamSnapshot, error) {
	var dao SubmissionDAO
	if err := r.db.WithContext(ctx).
		Model(&SubmissionDAO{}).
		Select("id", "user_id", "current_attempt_id", "status", "updated_at").
		First(&dao, submissionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSubmissionNotFound
		}
		return nil, fmt.Errorf("get submission stream snapshot %d: %w", submissionID, err)
	}

	return &entity.SubmissionStreamSnapshot{
		SubmissionID: dao.ID,
		UserID:       dao.UserID,
		AttemptID:    dao.CurrentAttemptID,
		Status:       entity.Status(dao.Status),
		UpdatedAt:    dao.UpdatedAt,
	}, nil
}

func (r *submissionRepository) getByID(ctx context.Context, id int64, forUpdate bool) (*entity.Submission, error) {
	var dao SubmissionDAO
	db := getDB(ctx, r.db)
	if forUpdate {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := db.First(&dao, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSubmissionNotFound
		}
		return nil, fmt.Errorf("get submission %d: %w", id, err)
	}

	return toSubmissionEntity(&dao), nil
}

func (r *submissionRepository) Update(ctx context.Context, submission *entity.Submission) error {
	updates := map[string]interface{}{
		"status":             string(submission.Status),
		"current_attempt_id": submission.CurrentAttemptID,
		"execution_time":     submission.ExecutionTime,
		"memory_used":        submission.MemoryUsed,
		"compile_output":     submission.CompileOutput,
		"error_message":      submission.ErrorMessage,
		"updated_at":         submission.UpdatedAt,
	}

	db := getDB(ctx, r.db)
	result := db.Model(&SubmissionDAO{}).Where("id = ?", submission.ID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update submission %d: %w", submission.ID, result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrSubmissionNotFound
	}

	return nil
}

func (r *submissionRepository) GetUserProfileStats(
	ctx context.Context,
	userID string,
	activitySince time.Time,
) (outbound.UserProfileStats, error) {
	type summaryRow struct {
		TotalSubmissions    int64
		AttemptedProblems   int64
		AcceptedSubmissions int64
		SolvedProblems      int64
	}
	type verdictRow struct {
		Verdict string
		Count   int64
	}
	type languageRow struct {
		Language string
		Count    int64
	}
	type activityRow struct {
		Date  string
		Count int64
	}

	terminalStatuses := []string{
		string(entity.StatusAccepted),
		string(entity.StatusWrongAnswer),
		string(entity.StatusTimeLimitExceed),
		string(entity.StatusMemoryLimitExceed),
		string(entity.StatusOutputLimitExceed),
		string(entity.StatusRuntimeError),
		string(entity.StatusCompilationError),
		string(entity.StatusSystemError),
	}

	var summary summaryRow
	if err := getDB(ctx, r.db).Model(&SubmissionDAO{}).Select(
		"COUNT(*) AS total_submissions, "+
			"COUNT(DISTINCT problem_id) AS attempted_problems, "+
			"COUNT(*) FILTER (WHERE status = ?) AS accepted_submissions, "+
			"COUNT(DISTINCT problem_id) FILTER (WHERE status = ?) AS solved_problems",
		string(entity.StatusAccepted),
		string(entity.StatusAccepted),
	).Where("user_id = ?", userID).Scan(&summary).Error; err != nil {
		return outbound.UserProfileStats{}, fmt.Errorf("get user profile stats summary: %w", err)
	}

	var verdictRows []verdictRow
	if err := getDB(ctx, r.db).Model(&SubmissionDAO{}).Select("status AS verdict, COUNT(*) AS count").
		Where("user_id = ? AND status IN ?", userID, terminalStatuses).
		Group("status").
		Order("count DESC, status ASC").
		Scan(&verdictRows).Error; err != nil {
		return outbound.UserProfileStats{}, fmt.Errorf("get user profile stats verdict distribution: %w", err)
	}

	var languageRows []languageRow
	if err := getDB(ctx, r.db).Model(&SubmissionDAO{}).Select("language, COUNT(*) AS count").
		Where("user_id = ?", userID).
		Group("language").
		Order("count DESC, language ASC").
		Scan(&languageRows).Error; err != nil {
		return outbound.UserProfileStats{}, fmt.Errorf("get user profile stats language distribution: %w", err)
	}

	const utcDay = "(created_at AT TIME ZONE 'UTC')::date"
	var activityRows []activityRow
	if err := getDB(ctx, r.db).Model(&SubmissionDAO{}).Select("TO_CHAR("+utcDay+", 'YYYY-MM-DD') AS date, COUNT(*) AS count").
		Where("user_id = ? AND created_at >= ?", userID, activitySince).
		Group(utcDay).
		Order(utcDay + " ASC").
		Scan(&activityRows).Error; err != nil {
		return outbound.UserProfileStats{}, fmt.Errorf("get user profile stats activity: %w", err)
	}

	stats := outbound.UserProfileStats{
		TotalSubmissions:    summary.TotalSubmissions,
		AttemptedProblems:   summary.AttemptedProblems,
		AcceptedSubmissions: summary.AcceptedSubmissions,
		SolvedProblems:      summary.SolvedProblems,
		Verdicts:            make([]outbound.ProfileStatsVerdict, 0, len(verdictRows)),
		Languages:           make([]outbound.ProfileStatsLanguage, 0, len(languageRows)),
		Activity:            make([]outbound.ProfileStatsActivity, 0, len(activityRows)),
	}
	for _, row := range verdictRows {
		stats.Verdicts = append(stats.Verdicts, outbound.ProfileStatsVerdict{Verdict: row.Verdict, Count: row.Count})
	}
	for _, row := range languageRows {
		stats.Languages = append(stats.Languages, outbound.ProfileStatsLanguage{Language: row.Language, Count: row.Count})
	}
	for _, row := range activityRows {
		stats.Activity = append(stats.Activity, outbound.ProfileStatsActivity{Date: row.Date, Count: row.Count})
	}
	return stats, nil
}

func (r *submissionRepository) List(
	ctx context.Context,
	filter outbound.ListSubmissionsFilter,
) (outbound.ListSubmissionsResult, error) {
	var total int64
	countQuery := applyListFilters(
		r.db.WithContext(ctx).Model(&SubmissionDAO{}),
		filter,
	)
	if err := countQuery.Count(&total).Error; err != nil {
		return outbound.ListSubmissionsResult{}, fmt.Errorf("count submissions: %w", err)
	}

	var daos []SubmissionDAO
	itemQuery := applyListFilters(
		r.db.WithContext(ctx).Model(&SubmissionDAO{}),
		filter,
	)
	if err := itemQuery.
		Select("id", "problem_id", "problem_name", "user_id", "username", "language", "status", "execution_time", "memory_used", "created_at").
		Order("created_at DESC, id DESC").
		Offset(filter.Offset).
		Limit(filter.Limit).
		Find(&daos).Error; err != nil {
		return outbound.ListSubmissionsResult{}, fmt.Errorf("list submissions: %w", err)
	}

	return outbound.ListSubmissionsResult{
		Items: toSubmissionEntities(daos),
		Total: total,
	}, nil
}

func (r *submissionRepository) ResultSummaries(
	ctx context.Context,
	submissionIDs []int64,
) (map[int64]outbound.SubmissionResultSummary, error) {
	if len(submissionIDs) == 0 {
		return map[int64]outbound.SubmissionResultSummary{}, nil
	}

	type resultSummaryRow struct {
		SubmissionID int64
		Passed       int
		Total        int
	}

	var rows []resultSummaryRow
	if err := r.db.WithContext(ctx).
		Model(&SubmissionResultDAO{}).
		Select(
			"submission_results.submission_id, "+
				"SUM(CASE WHEN submission_results.status = ? THEN 1 ELSE 0 END) AS passed, "+
				"COUNT(*) AS total",
			string(entity.ResultAccepted),
		).
		Joins("JOIN submissions ON submissions.id = submission_results.submission_id").
		Where("submission_results.submission_id IN ? AND submission_results.attempt_id = submissions.current_attempt_id", submissionIDs).
		Group("submission_results.submission_id").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("summarize submission results: %w", err)
	}

	summaries := make(map[int64]outbound.SubmissionResultSummary, len(rows))
	for _, row := range rows {
		summaries[row.SubmissionID] = outbound.SubmissionResultSummary{
			SubmissionID: row.SubmissionID,
			Passed:       row.Passed,
			Total:        row.Total,
		}
	}
	return summaries, nil
}

func applyListFilters(
	query *gorm.DB,
	filter outbound.ListSubmissionsFilter,
) *gorm.DB {
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.Language != nil {
		query = query.Where("language = ?", *filter.Language)
	}
	if filter.ProblemID != nil {
		query = query.Where("problem_id = ?", *filter.ProblemID)
	}
	return query
}

func toSubmissionDAO(s *entity.Submission) *SubmissionDAO {
	return &SubmissionDAO{
		ID:               s.ID,
		ProblemID:        s.ProblemID,
		ProblemName:      s.ProblemName,
		UserID:           s.UserID,
		Username:         s.Username,
		Language:         string(s.Language),
		SourceCode:       s.SourceCode,
		CurrentAttemptID: s.CurrentAttemptID,
		Status:           string(s.Status),
		ExecutionTime:    s.ExecutionTime,
		MemoryUsed:       s.MemoryUsed,
		CompileOutput:    s.CompileOutput,
		ErrorMessage:     s.ErrorMessage,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}

func toSubmissionEntity(dao *SubmissionDAO) *entity.Submission {
	return &entity.Submission{
		ID:               dao.ID,
		ProblemID:        dao.ProblemID,
		ProblemName:      dao.ProblemName,
		UserID:           dao.UserID,
		Username:         dao.Username,
		Language:         entity.Language(dao.Language),
		SourceCode:       dao.SourceCode,
		CurrentAttemptID: dao.CurrentAttemptID,
		Status:           entity.Status(dao.Status),
		ExecutionTime:    dao.ExecutionTime,
		MemoryUsed:       dao.MemoryUsed,
		CompileOutput:    dao.CompileOutput,
		ErrorMessage:     dao.ErrorMessage,
		CreatedAt:        dao.CreatedAt,
		UpdatedAt:        dao.UpdatedAt,
	}
}

func toSubmissionEntities(daos []SubmissionDAO) []*entity.Submission {
	results := make([]*entity.Submission, 0, len(daos))
	for i := range daos {
		results = append(results, toSubmissionEntity(&daos[i]))
	}
	return results
}
