package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
)

type fakePublicUserResolver struct {
	user     outbound.PublicUser
	err      error
	username string
	calls    int
}

func (r *fakePublicUserResolver) ResolvePublicUser(_ context.Context, username string) (outbound.PublicUser, error) {
	r.calls++
	r.username = username
	return r.user, r.err
}

func TestGetPublicProfileStatsResolvesAuthOwnedIdentityBeforeSubmissionStats(t *testing.T) {
	resolver := &fakePublicUserResolver{user: outbound.PublicUser{ID: "user-1", Username: "ada"}}
	repo := &fakeProfileStatsRepository{stats: outbound.UserProfileStats{TotalSubmissions: 4, AttemptedProblems: 3, AcceptedSubmissions: 2, SolvedProblems: 2}}
	uc := NewGetPublicProfileStatsUseCase(resolver, repo).(*getPublicProfileStatsUseCase)
	uc.now = func() time.Time { return time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC) }

	got, err := uc.Execute(context.Background(), dto.GetPublicProfileStatsRequest{Username: "ada"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if resolver.calls != 1 || resolver.username != "ada" || repo.calls != 1 || repo.userID != "user-1" {
		t.Fatalf("resolver/repository = %+v/%+v", resolver, repo)
	}
	if got.SolvedProblems != 2 || got.AcceptanceRate != 50 || got.Activity == nil || got.LanguageDistribution == nil || got.VerdictDistribution == nil {
		t.Fatalf("response = %+v", got)
	}
}

func TestGetPublicProfileStatsDoesNotQuerySubmissionDataWhenAuthRejectsOrFails(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want error
	}{
		{name: "not found", err: domain.ErrPublicProfileNotFound, want: domain.ErrPublicProfileNotFound},
		{name: "auth unavailable", err: domain.ErrAuthServiceUnavailable, want: domain.ErrAuthServiceUnavailable},
		{name: "canceled", err: context.Canceled, want: context.Canceled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &fakePublicUserResolver{err: tt.err}
			repo := &fakeProfileStatsRepository{}
			_, err := NewGetPublicProfileStatsUseCase(resolver, repo).Execute(context.Background(), dto.GetPublicProfileStatsRequest{Username: "ada"})
			if !errors.Is(err, tt.want) || repo.calls != 0 {
				t.Fatalf("error/repository calls = %v/%d, want %v/0", err, repo.calls, tt.want)
			}
		})
	}
}

func TestGetPublicProfileStatsMapsSubmissionRepositoryFailure(t *testing.T) {
	resolver := &fakePublicUserResolver{user: outbound.PublicUser{ID: "user-1"}}
	_, err := NewGetPublicProfileStatsUseCase(resolver, &fakeProfileStatsRepository{err: errors.New("database unavailable")}).Execute(
		context.Background(), dto.GetPublicProfileStatsRequest{Username: "ada"},
	)
	if !errors.Is(err, domain.ErrInternalServer) {
		t.Fatalf("repository error = %v, want internal", err)
	}
}
