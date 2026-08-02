package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"
)

type fakeIssueTicketSnapshotRepository struct {
	snapshot *entity.SubmissionStreamSnapshot
	err      error
	calls    int
	id       int64
}

func (r *fakeIssueTicketSnapshotRepository) GetStreamSnapshot(
	ctx context.Context,
	submissionID int64,
) (*entity.SubmissionStreamSnapshot, error) {
	r.calls++
	r.id = submissionID
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return r.snapshot, r.err
}

type fakeIssueTicketService struct {
	ticket       string
	expiresAt    time.Time
	err          error
	issueCalls   int
	userID       string
	submissionID int64
}

func (s *fakeIssueTicketService) Issue(userID string, submissionID int64) (string, time.Time, error) {
	s.issueCalls++
	s.userID = userID
	s.submissionID = submissionID
	return s.ticket, s.expiresAt, s.err
}

func (*fakeIssueTicketService) Verify(string) (entity.SubmissionStreamTicketClaims, error) {
	return entity.SubmissionStreamTicketClaims{}, nil
}

func TestIssueSubmissionStreamTicketSuccess(t *testing.T) {
	expiresAt := time.Date(2026, 7, 30, 9, 2, 0, 0, time.UTC)
	snapshotRepo := &fakeIssueTicketSnapshotRepository{
		snapshot: &entity.SubmissionStreamSnapshot{
			SubmissionID: 77,
			UserID:       "owner",
			AttemptID:    "attempt-77",
			Status:       entity.StatusJudging,
		},
	}
	ticketService := &fakeIssueTicketService{ticket: "opaque-ticket", expiresAt: expiresAt}

	got, err := NewIssueSubmissionStreamTicketUseCase(snapshotRepo, ticketService).Execute(
		context.Background(),
		auth.Claims{UserID: "owner", Role: rbac.RoleUser},
		dto.IssueSubmissionStreamTicketRequest{SubmissionID: 77},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.Ticket != "opaque-ticket" || !got.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("response = %+v", got)
	}
	if snapshotRepo.calls != 1 || snapshotRepo.id != 77 {
		t.Fatalf("snapshot repo calls/id = %d/%d", snapshotRepo.calls, snapshotRepo.id)
	}
	if ticketService.issueCalls != 1 || ticketService.userID != "owner" || ticketService.submissionID != 77 {
		t.Fatalf("ticket issue = calls %d user %q submission %d", ticketService.issueCalls, ticketService.userID, ticketService.submissionID)
	}
}

func TestIssueSubmissionStreamTicketRejectsInvalidActorAndRequest(t *testing.T) {
	tests := []struct {
		name    string
		claims  auth.Claims
		req     dto.IssueSubmissionStreamTicketRequest
		wantErr error
	}{
		{name: "invalid submission id", claims: auth.Claims{UserID: "owner", Role: rbac.RoleUser}, wantErr: domain.ErrInvalidSubmissionID},
		{name: "missing user id", claims: auth.Claims{Role: rbac.RoleUser}, req: dto.IssueSubmissionStreamTicketRequest{SubmissionID: 77}, wantErr: domain.ErrSubmissionUnauthenticated},
		{name: "missing role", claims: auth.Claims{UserID: "owner"}, req: dto.IssueSubmissionStreamTicketRequest{SubmissionID: 77}, wantErr: domain.ErrSubmissionUnauthenticated},
		{name: "unsupported role", claims: auth.Claims{UserID: "owner", Role: rbac.Role("guest")}, req: dto.IssueSubmissionStreamTicketRequest{SubmissionID: 77}, wantErr: domain.ErrSubmissionForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshotRepo := &fakeIssueTicketSnapshotRepository{}
			ticketService := &fakeIssueTicketService{}
			_, err := NewIssueSubmissionStreamTicketUseCase(snapshotRepo, ticketService).Execute(
				context.Background(),
				tt.claims,
				tt.req,
			)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}
			if snapshotRepo.calls != 0 || ticketService.issueCalls != 0 {
				t.Fatalf("unexpected calls: repo=%d ticket=%d", snapshotRepo.calls, ticketService.issueCalls)
			}
		})
	}
}

func TestIssueSubmissionStreamTicketRejectsOtherOwner(t *testing.T) {
	snapshotRepo := &fakeIssueTicketSnapshotRepository{
		snapshot: &entity.SubmissionStreamSnapshot{SubmissionID: 77, UserID: "owner"},
	}
	ticketService := &fakeIssueTicketService{}

	_, err := NewIssueSubmissionStreamTicketUseCase(snapshotRepo, ticketService).Execute(
		context.Background(),
		auth.Claims{UserID: "attacker", Role: rbac.RoleAdmin},
		dto.IssueSubmissionStreamTicketRequest{SubmissionID: 77},
	)
	if !errors.Is(err, domain.ErrSubmissionNotFound) {
		t.Fatalf("Execute() error = %v, want not found", err)
	}
	if ticketService.issueCalls != 0 {
		t.Fatalf("ticket issue calls = %d, want 0", ticketService.issueCalls)
	}
}

func TestIssueSubmissionStreamTicketRepositoryAndTicketErrors(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	ticketErr := errors.New("hmac unavailable")

	tests := []struct {
		name      string
		repo      *fakeIssueTicketSnapshotRepository
		ticketSvc *fakeIssueTicketService
		wantErr   error
	}{
		{
			name:    "not found",
			repo:    &fakeIssueTicketSnapshotRepository{err: domain.ErrSubmissionNotFound},
			wantErr: domain.ErrSubmissionNotFound,
		},
		{
			name:    "repository failure",
			repo:    &fakeIssueTicketSnapshotRepository{err: databaseErr},
			wantErr: domain.ErrInternalServer,
		},
		{
			name: "ticket failure",
			repo: &fakeIssueTicketSnapshotRepository{
				snapshot: &entity.SubmissionStreamSnapshot{SubmissionID: 77, UserID: "owner"},
			},
			ticketSvc: &fakeIssueTicketService{err: ticketErr},
			wantErr:   domain.ErrInternalServer,
		},
		{
			name:    "nil snapshot",
			repo:    &fakeIssueTicketSnapshotRepository{},
			wantErr: domain.ErrInternalServer,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticketSvc := tt.ticketSvc
			if ticketSvc == nil {
				ticketSvc = &fakeIssueTicketService{}
			}
			_, err := NewIssueSubmissionStreamTicketUseCase(tt.repo, ticketSvc).Execute(
				context.Background(),
				auth.Claims{UserID: "owner", Role: rbac.RoleUser},
				dto.IssueSubmissionStreamTicketRequest{SubmissionID: 77},
			)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestIssueSubmissionStreamTicketPreservesContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewIssueSubmissionStreamTicketUseCase(
		&fakeIssueTicketSnapshotRepository{snapshot: &entity.SubmissionStreamSnapshot{SubmissionID: 77, UserID: "owner"}},
		&fakeIssueTicketService{},
	).Execute(
		ctx,
		auth.Claims{UserID: "owner", Role: rbac.RoleUser},
		dto.IssueSubmissionStreamTicketRequest{SubmissionID: 77},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Execute() error = %v, want context canceled", err)
	}
}
