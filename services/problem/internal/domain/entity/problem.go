package entity

import "time"

type Difficulty string

const (
	Easy   Difficulty = "EASY"
	Medium Difficulty = "MEDIUM"
	Hard   Difficulty = "HARD"
)

type Problem struct {
	ID          int64
	Slug        string
	Title       string
	Description string
	Difficulty  Difficulty

	TimeLimit   int
	MemoryLimit int

	AuthorID string
	IsHidden bool

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func NewProblem(title, slug, desc string, diff Difficulty, timeLimit, memLimit int, authorIDs string) *Problem {
	return &Problem{
		Title:       title,
		Slug:        slug,
		Description: desc,
		Difficulty:  diff,
		TimeLimit:   timeLimit,
		MemoryLimit: memLimit,
		AuthorID:    authorIDs,
		IsHidden:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func (p *Problem) Publish() {
	p.IsHidden = false
	p.UpdatedAt = time.Now()
}

func (p *Problem) Hide() {
	p.IsHidden = true
	p.UpdatedAt = time.Now()
}

func (p *Problem) IsDeleted() bool {
	return p.DeletedAt != nil
}
