package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
)

type fakeProfileStatsRepository struct {
	stats         outbound.UserProfileStats
	err           error
	userID        string
	activitySince time.Time
	calls         int
}

func (r *fakeProfileStatsRepository) GetUserProfileStats(
	ctx context.Context,
	userID string,
	activitySince time.Time,
) (outbound.UserProfileStats, error) {
	r.calls++
	r.userID = userID
	r.activitySince = activitySince
	if err := ctx.Err(); err != nil {
		return outbound.UserProfileStats{}, err
	}
	return r.stats, r.err
}

func TestGetMyProfileStatsReturnsAuthoritativeRepositoryAggregates(t *testing.T) {
	repo := &fakeProfileStatsRepository{stats: outbound.UserProfileStats{
		TotalSubmissions:    4,
		AttemptedProblems:   3,
		AcceptedSubmissions: 2,
		SolvedProblems:      2,
		Verdicts: []outbound.ProfileStatsVerdict{
			{Verdict: "ACCEPTED", Count: 2},
			{Verdict: "WRONG_ANSWER", Count: 1},
		},
		Languages: []outbound.ProfileStatsLanguage{
			{Language: "GO", Count: 3},
			{Language: "CPP", Count: 1},
		},
		Activity: []outbound.ProfileStatsActivity{{Date: "2026-08-12", Count: 3}},
	}}
	uc := NewGetMyProfileStatsUseCase(repo).(*getMyProfileStatsUseCase)
	uc.now = func() time.Time { return time.Date(2026, 8, 12, 15, 0, 0, 0, time.FixedZone("UTC+7", 7*60*60)) }

	got, err := uc.Execute(context.Background(), auth.Claims{UserID: "actor", Role: rbac.RoleUser})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repo.calls != 1 || repo.userID != "actor" {
		t.Fatalf("repository calls/user = %d/%q, want 1/actor", repo.calls, repo.userID)
	}
	wantSince := time.Date(2025, 8, 13, 0, 0, 0, 0, time.UTC)
	if !repo.activitySince.Equal(wantSince) {
		t.Fatalf("activity since = %s, want %s", repo.activitySince, wantSince)
	}
	if got.TotalSubmissions != 4 || got.AttemptedProblems != 3 || got.AcceptedSubmissions != 2 || got.SolvedProblems != 2 || got.AcceptanceRate != 50 {
		t.Fatalf("summary = %+v", got)
	}
	if len(got.VerdictDistribution) != 2 || got.VerdictDistribution[0].Verdict != "ACCEPTED" ||
		len(got.LanguageDistribution) != 2 || got.LanguageDistribution[0].Language != "GO" ||
		len(got.Activity) != 1 || got.Activity[0].Date != "2026-08-12" {
		t.Fatalf("distributions/activity = %+v", got)
	}
}

func TestGetMyProfileStatsZeroAndAuthorizationBehavior(t *testing.T) {
	for _, role := range []rbac.Role{rbac.RoleUser, rbac.RoleContributor, rbac.RoleModerator, rbac.RoleAdmin} {
		t.Run(string(role), func(t *testing.T) {
			repo := &fakeProfileStatsRepository{}
			got, err := NewGetMyProfileStatsUseCase(repo).Execute(context.Background(), auth.Claims{UserID: "actor", Role: role})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if got.TotalSubmissions != 0 || got.AttemptedProblems != 0 || got.AcceptedSubmissions != 0 || got.SolvedProblems != 0 || got.AcceptanceRate != 0 ||
				got.VerdictDistribution == nil || got.LanguageDistribution == nil || got.Activity == nil {
				t.Fatalf("zero response = %+v", got)
			}
		})
	}

	for _, tt := range []struct {
		name   string
		claims auth.Claims
		want   error
	}{
		{name: "missing actor", claims: auth.Claims{Role: rbac.RoleUser}, want: domain.ErrSubmissionUnauthenticated},
		{name: "unknown role", claims: auth.Claims{UserID: "actor", Role: rbac.Role("auditor")}, want: domain.ErrSubmissionForbidden},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeProfileStatsRepository{}
			_, err := NewGetMyProfileStatsUseCase(repo).Execute(context.Background(), tt.claims)
			if !errors.Is(err, tt.want) || repo.calls != 0 {
				t.Fatalf("error/calls = %v/%d, want %v/0", err, repo.calls, tt.want)
			}
		})
	}
}

func TestGetMyProfileStatsPreservesCurrentAttemptAggregateSemantics(t *testing.T) {
	for _, tt := range []struct {
		name     string
		stats    outbound.UserProfileStats
		wantRate float64
	}{
		{
			name:     "basic statistics",
			stats:    outbound.UserProfileStats{TotalSubmissions: 4, AttemptedProblems: 3, AcceptedSubmissions: 2, SolvedProblems: 2},
			wantRate: 50,
		},
		{
			name:     "many submissions to one problem count as one attempted problem",
			stats:    outbound.UserProfileStats{TotalSubmissions: 4, AttemptedProblems: 1, AcceptedSubmissions: 1, SolvedProblems: 1},
			wantRate: 25,
		},
		{
			name:     "multiple distinct problems including pending are attempted",
			stats:    outbound.UserProfileStats{TotalSubmissions: 3, AttemptedProblems: 3, AcceptedSubmissions: 0, SolvedProblems: 0},
			wantRate: 0,
		},
		{
			name:     "multiple users remain scoped to the authenticated user",
			stats:    outbound.UserProfileStats{TotalSubmissions: 2, AttemptedProblems: 2, AcceptedSubmissions: 0, SolvedProblems: 0},
			wantRate: 0,
		},
		{
			name:     "multiple accepted submissions on one problem",
			stats:    outbound.UserProfileStats{TotalSubmissions: 2, AttemptedProblems: 1, AcceptedSubmissions: 2, SolvedProblems: 1},
			wantRate: 100,
		},
		{
			name:     "accepted followed by wrong answer remains solved",
			stats:    outbound.UserProfileStats{TotalSubmissions: 2, AttemptedProblems: 1, AcceptedSubmissions: 1, SolvedProblems: 1},
			wantRate: 50,
		},
		{
			name:     "rejudge accepted to wrong answer retains attempted problem",
			stats:    outbound.UserProfileStats{TotalSubmissions: 1, AttemptedProblems: 1, AcceptedSubmissions: 0, SolvedProblems: 0},
			wantRate: 0,
		},
		{
			name:     "rejudge wrong answer to accepted counts current acceptance",
			stats:    outbound.UserProfileStats{TotalSubmissions: 1, AttemptedProblems: 1, AcceptedSubmissions: 1, SolvedProblems: 1},
			wantRate: 100,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeProfileStatsRepository{stats: tt.stats}
			got, err := NewGetMyProfileStatsUseCase(repo).Execute(
				context.Background(),
				auth.Claims{UserID: "user-a", Role: rbac.RoleUser},
			)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if got.TotalSubmissions != tt.stats.TotalSubmissions ||
				got.AttemptedProblems != tt.stats.AttemptedProblems ||
				got.AcceptedSubmissions != tt.stats.AcceptedSubmissions ||
				got.SolvedProblems != tt.stats.SolvedProblems ||
				got.SolvedProblems > got.AttemptedProblems || got.AcceptanceRate != tt.wantRate || repo.userID != "user-a" {
				t.Fatalf("response/repository user = %+v/%q", got, repo.userID)
			}
		})
	}
}

func TestGetMyProfileStatsMapsRepositoryErrors(t *testing.T) {
	for _, wantErr := range []error{errors.New("database unavailable"), context.Canceled} {
		repo := &fakeProfileStatsRepository{err: wantErr}
		_, err := NewGetMyProfileStatsUseCase(repo).Execute(context.Background(), auth.Claims{UserID: "actor", Role: rbac.RoleUser})
		if wantErr == context.Canceled {
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %v, want canceled", err)
			}
			continue
		}
		if !errors.Is(err, domain.ErrInternalServer) {
			t.Fatalf("error = %v, want internal server", err)
		}
	}
}

func TestAcceptanceRateRoundsToTwoDecimalPlaces(t *testing.T) {
	if got := acceptanceRate(1, 3); got != 33.33 {
		t.Fatalf("rate = %v, want 33.33", got)
	}
}
