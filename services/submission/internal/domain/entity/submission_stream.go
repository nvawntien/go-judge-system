package entity

import "time"

const SubmissionStreamTicketPurpose = "submission_sse"

type SubmissionStreamTicketClaims struct {
	Purpose      string    `json:"purpose"`
	UserID       string    `json:"user_id"`
	SubmissionID int64     `json:"submission_id"`
	IssuedAt     time.Time `json:"issued_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type SubmissionStreamSnapshot struct {
	SubmissionID int64     `json:"submission_id"`
	UserID       string    `json:"-"`
	AttemptID    string    `json:"attempt_id"`
	Status       Status    `json:"status"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type SubmissionEvent struct {
	SubmissionID int64     `json:"submission_id"`
	AttemptID    string    `json:"attempt_id"`
	Status       string    `json:"status"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (s SubmissionStreamSnapshot) Event() SubmissionEvent {
	return SubmissionEvent{
		SubmissionID: s.SubmissionID,
		AttemptID:    s.AttemptID,
		Status:       string(s.Status),
		UpdatedAt:    s.UpdatedAt,
	}
}
