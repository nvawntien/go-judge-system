package entity

import "time"

type Difficulty string

const (
	Easy   Difficulty = "easy"
	Medium Difficulty = "medium"
	Hard   Difficulty = "hard"
)

// ProblemExample represents a sample test case displayed to users on the problem page.
type ProblemExample struct {
	Input       string
	Output      string
	Explanation string
}

type Problem struct {
	ID          int64
	Title       string
	TitleSlug   string
	Description string
	Difficulty  Difficulty
	Tags        []Tag

	Examples    []ProblemExample
	Constraints []string
	Hints       []string

	TimeLimit   float64
	MemoryLimit int

	AuthorID string
	IsHidden bool

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func NewProblem(
	title, slug, desc string,
	diff Difficulty,
	tags []Tag,
	examples []ProblemExample,
	constraints []string,
	hints []string,
	timeLimit float64,
	memLimit int,
	authorID string,
) *Problem {
	return &Problem{
		Title:       title,
		TitleSlug:   slug,
		Description: desc,
		Difficulty:  diff,
		Tags:        tags,
		Examples:    examples,
		Constraints: constraints,
		Hints:       hints,
		TimeLimit:   timeLimit,
		MemoryLimit: memLimit,
		AuthorID:    authorID,
		IsHidden:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func (p *Problem) Publish() {
	p.IsHidden = false
	p.UpdatedAt = time.Now()
}

func (p *Problem) Hidden() {
	p.IsHidden = true
	p.UpdatedAt = time.Now()
}

func (p *Problem) IsDeleted() bool {
	return p.DeletedAt != nil
}
