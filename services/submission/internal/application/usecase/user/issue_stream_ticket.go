package user

import (
	"context"
	"errors"
	"strings"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/submission/internal/application/dto"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/user"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
)

type issueSubmissionStreamTicketUseCase struct {
	snapshotRepo  outbound.SubmissionStreamSnapshotRepository
	ticketService outbound.SubmissionStreamTicketService
}

func NewIssueSubmissionStreamTicketUseCase(
	snapshotRepo outbound.SubmissionStreamSnapshotRepository,
	ticketService outbound.SubmissionStreamTicketService,
) inbound.IssueSubmissionStreamTicketUseCase {
	return &issueSubmissionStreamTicketUseCase{
		snapshotRepo:  snapshotRepo,
		ticketService: ticketService,
	}
}

func (uc *issueSubmissionStreamTicketUseCase) Execute(
	ctx context.Context,
	claims auth.Claims,
	req dto.IssueSubmissionStreamTicketRequest,
) (dto.IssueSubmissionStreamTicketResponse, error) {
	if req.SubmissionID <= 0 {
		return dto.IssueSubmissionStreamTicketResponse{}, domain.ErrInvalidSubmissionID
	}
	if strings.TrimSpace(claims.UserID) == "" || claims.Role == "" {
		return dto.IssueSubmissionStreamTicketResponse{}, domain.ErrSubmissionUnauthenticated
	}

	switch claims.Role {
	case rbac.RoleUser, rbac.RoleContributor, rbac.RoleModerator, rbac.RoleAdmin:
	default:
		return dto.IssueSubmissionStreamTicketResponse{}, domain.ErrSubmissionForbidden
	}

	snapshot, err := uc.snapshotRepo.GetStreamSnapshot(ctx, req.SubmissionID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSubmissionNotFound):
			return dto.IssueSubmissionStreamTicketResponse{}, domain.ErrSubmissionNotFound
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return dto.IssueSubmissionStreamTicketResponse{}, err
		default:
			return dto.IssueSubmissionStreamTicketResponse{}, domain.ErrInternalServer.Wrap(err)
		}
	}
	if snapshot == nil {
		return dto.IssueSubmissionStreamTicketResponse{}, domain.ErrInternalServer
	}
	if snapshot.UserID != claims.UserID {
		return dto.IssueSubmissionStreamTicketResponse{}, domain.ErrSubmissionNotFound
	}

	ticket, expiresAt, err := uc.ticketService.Issue(claims.UserID, snapshot.SubmissionID)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return dto.IssueSubmissionStreamTicketResponse{}, err
		}
		return dto.IssueSubmissionStreamTicketResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	return dto.IssueSubmissionStreamTicketResponse{
		Ticket:    ticket,
		ExpiresAt: expiresAt,
	}, nil
}
