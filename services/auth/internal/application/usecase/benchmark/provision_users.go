// Package benchmark provisions the fixed, non-public benchmark account pool.
package benchmark

import (
	"context"
	"errors"
	"fmt"

	"go-judge-system/pkg/rbac"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"
	"go-judge-system/services/auth/internal/domain/valueobject"
)

const (
	MinSequence = 1
	// MaxSequence deliberately bounds this operator-only fixture namespace. The
	// %03d identity format remains stable for 001..100 and naturally extends to
	// 1000..100000 without renaming existing users.
	MaxSequence = 100000
)

var (
	ErrInvalidRange  = errors.New("invalid benchmark user range")
	ErrConflicts     = errors.New("benchmark user plan contains conflicts")
	ErrApplyStopped  = errors.New("benchmark user provisioning stopped safely")
	ErrRotateStopped = errors.New("benchmark password rotation stopped safely")
)

type Identity struct {
	Sequence int
	Username string
	Email    string
	FullName string
}

type Status string

const (
	StatusWouldCreate Status = "would_create"
	StatusExisting    Status = "existing_identity"
	StatusCreated     Status = "created"
	StatusSkipped     Status = "skipped"
	StatusConflict    Status = "conflict"
	StatusWouldRotate Status = "would_rotate"
	StatusRotated     Status = "rotated"
)

type Entry struct {
	Identity
	Status Status
	Reason string
}

type Request struct {
	Start    int
	Count    int
	Apply    bool
	Password []byte
	// Progress is an optional process-local callback. It receives only counts
	// and is deliberately not persisted or used by the domain/repository.
	Progress func(completed, total int)
}

type Result struct {
	Entries   []Entry
	Created   int
	Skipped   int
	Existing  int
	Conflicts int
	StoppedAt *Identity
	Rotated   int
}

type Provisioner struct {
	users     outbound.UserRepository
	passwords outbound.PasswordEncoder
}

func NewProvisioner(users outbound.UserRepository, passwords outbound.PasswordEncoder) *Provisioner {
	return &Provisioner{users: users, passwords: passwords}
}

func Identities(start, count int) ([]Identity, error) {
	if start < MinSequence || count < 1 || start > MaxSequence || count-1 > MaxSequence-start {
		return nil, ErrInvalidRange
	}

	identities := make([]Identity, 0, count)
	for sequence := start; sequence < start+count; sequence++ {
		identities = append(identities, Identity{
			Sequence: sequence,
			Username: fmt.Sprintf("benchmark_judge_%03d", sequence),
			Email:    fmt.Sprintf("benchmark-judge-%03d@benchmark.invalid", sequence),
			FullName: fmt.Sprintf("Benchmark Judge %03d", sequence),
		})
	}
	return identities, nil
}

// Plan inspects the full deterministic range without invoking the password
// encoder or mutating the repository. It is the Phase A safety gate for apply.
func (p *Provisioner) Plan(ctx context.Context, start, count int) (Result, error) {
	identities, err := Identities(start, count)
	if err != nil {
		return Result{}, err
	}

	result := Result{Entries: make([]Entry, 0, len(identities))}
	for _, identity := range identities {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		entry, _, err := p.inspect(ctx, identity)
		if err != nil {
			return result, err
		}
		result.Entries = append(result.Entries, entry)
	}
	result.recount()
	return result, nil
}

// PlanRotation validates that every requested fixture already exists and is
// canonical. Unlike normal provisioning, a missing identity is a conflict.
func (p *Provisioner) PlanRotation(ctx context.Context, start, count int) (Result, error) {
	identities, err := Identities(start, count)
	if err != nil {
		return Result{}, err
	}
	result := Result{Entries: make([]Entry, 0, len(identities))}
	for _, identity := range identities {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		entry, _, err := p.inspect(ctx, identity)
		if err != nil {
			return result, err
		}
		if entry.Status == StatusWouldCreate {
			entry.Status, entry.Reason = StatusConflict, "benchmark identity is missing"
		} else if entry.Status == StatusExisting {
			entry.Status = StatusWouldRotate
		}
		result.Entries = append(result.Entries, entry)
	}
	result.recount()
	return result, nil
}

// RotatePassword hashes the new password per account, then delegates the
// entire narrow update set to one adapter transaction. Existing password hashes
// are deliberately irrelevant in rotation mode.
func (p *Provisioner) RotatePassword(ctx context.Context, req Request) (Result, error) {
	if !req.Apply {
		return p.PlanRotation(ctx, req.Start, req.Count)
	}
	if len(req.Password) == 0 {
		return Result{}, errors.New("benchmark password is required for rotation")
	}
	if err := valueobject.ValidatePlainPassword(string(req.Password)); err != nil {
		return Result{}, err
	}
	result, err := p.PlanRotation(ctx, req.Start, req.Count)
	if err != nil {
		return result, err
	}
	if result.Conflicts != 0 {
		return result, ErrConflicts
	}
	rotator, ok := p.users.(outbound.BenchmarkPasswordRotator)
	if !ok {
		return result, ErrRotateStopped
	}
	updates := make([]outbound.BenchmarkPasswordUpdate, 0, len(result.Entries))
	for index := range result.Entries {
		if err := ctx.Err(); err != nil {
			result.StoppedAt = &result.Entries[index].Identity
			return result, err
		}
		entry := &result.Entries[index]
		fresh, user, err := p.inspect(ctx, entry.Identity)
		if err != nil || fresh.Status != StatusExisting || user == nil {
			entry.Status, entry.Reason, result.StoppedAt = StatusConflict, "benchmark identity changed before rotation", &entry.Identity
			result.recount()
			return result, ErrRotateStopped
		}
		hash, err := p.passwords.HashAndSalt(req.Password)
		if err != nil || ctx.Err() != nil {
			result.StoppedAt = &entry.Identity
			return result, ErrRotateStopped
		}
		updates = append(updates, outbound.BenchmarkPasswordUpdate{UserID: user.ID, Username: entry.Username, Email: entry.Email, FullName: entry.FullName, PasswordHash: hash})
		reportProgress(req, index+1)
	}
	if err := rotator.RotateBenchmarkPasswords(ctx, updates); err != nil {
		return result, ErrRotateStopped
	}
	for index := range result.Entries {
		result.Entries[index].Status = StatusRotated
	}
	result.recount()
	return result, nil
}

// Execute is Phase B. It re-plans to close the gap between planning and
// mutation, verifies existing hashes with the supplied password, and creates
// only after all currently visible conflicts are rejected.
func (p *Provisioner) Execute(ctx context.Context, req Request) (Result, error) {
	if !req.Apply {
		return p.Plan(ctx, req.Start, req.Count)
	}
	if len(req.Password) == 0 {
		return Result{}, fmt.Errorf("benchmark password is required for apply")
	}
	if err := valueobject.ValidatePlainPassword(string(req.Password)); err != nil {
		return Result{}, err
	}

	result, err := p.Plan(ctx, req.Start, req.Count)
	if err != nil {
		return result, err
	}
	for index := range result.Entries {
		entry := &result.Entries[index]
		if entry.Status != StatusExisting {
			continue
		}
		_, user, err := p.inspect(ctx, entry.Identity)
		if err != nil {
			return result, err
		}
		if !p.passwords.ComparePasswords(user.Password, req.Password) {
			entry.Status = StatusConflict
			entry.Reason = "existing canonical account password does not match"
		}
	}
	result.recount()
	if result.Conflicts != 0 {
		return result, ErrConflicts
	}

	for index := range result.Entries {
		if err := ctx.Err(); err != nil {
			result.StoppedAt = &result.Entries[index].Identity
			return result, err
		}
		entry := &result.Entries[index]
		if entry.Status == StatusExisting {
			entry.Status = StatusSkipped
			reportProgress(req, index+1)
			continue
		}

		// Re-read each candidate immediately before hashing/creation so a
		// concurrent provisioner cannot cause an unsafe overwrite.
		fresh, existing, err := p.inspect(ctx, entry.Identity)
		if err != nil {
			result.StoppedAt = &entry.Identity
			result.recount()
			return result, ErrApplyStopped
		}
		if fresh.Status == StatusExisting {
			if p.passwords.ComparePasswords(existing.Password, req.Password) {
				entry.Status = StatusSkipped
				continue
			}
			entry.Status = StatusConflict
			entry.Reason = "concurrent canonical account password does not match"
			result.StoppedAt = &entry.Identity
			result.recount()
			return result, ErrApplyStopped
		}
		if fresh.Status == StatusConflict {
			entry.Status = StatusConflict
			entry.Reason = fresh.Reason
			result.StoppedAt = &entry.Identity
			result.recount()
			return result, ErrApplyStopped
		}

		hash, err := p.passwords.HashAndSalt(req.Password)
		if err != nil || ctx.Err() != nil {
			result.StoppedAt = &entry.Identity
			result.recount()
			return result, ErrApplyStopped
		}
		email, err := valueobject.NewEmail(entry.Email)
		if err != nil {
			result.StoppedAt = &entry.Identity
			result.recount()
			return result, ErrApplyStopped
		}
		user := entity.NewUser(entry.FullName, entry.Username, email, valueobject.NewPasswordFromHash(hash))
		user.Activate()
		if err := p.users.CreateUser(ctx, user); err == nil {
			entry.Status = StatusCreated
			reportProgress(req, index+1)
			continue
		}

		// A database unique-constraint error is intentionally re-read because
		// this repository does not rely on global GORM error translation.
		reread, racedUser, rereadErr := p.inspect(ctx, entry.Identity)
		if rereadErr == nil && reread.Status == StatusExisting && p.passwords.ComparePasswords(racedUser.Password, req.Password) {
			entry.Status = StatusSkipped
			reportProgress(req, index+1)
			continue
		}
		entry.Status = StatusConflict
		entry.Reason = "create failed and identity could not be verified safely"
		result.StoppedAt = &entry.Identity
		result.recount()
		return result, ErrApplyStopped
	}

	result.recount()
	return result, nil
}

func reportProgress(req Request, completed int) {
	if req.Progress != nil {
		req.Progress(completed, req.Count)
	}
}

func (p *Provisioner) inspect(ctx context.Context, identity Identity) (Entry, *entity.User, error) {
	byUsername, usernameErr := p.users.GetUserByUsername(ctx, identity.Username)
	byEmail, emailErr := p.users.GetUserByEmail(ctx, identity.Email)

	if usernameErr != nil && !errors.Is(usernameErr, domain.ErrUserNotFound) {
		return Entry{}, nil, fmt.Errorf("inspect benchmark username")
	}
	if emailErr != nil && !errors.Is(emailErr, domain.ErrUserNotFound) {
		return Entry{}, nil, fmt.Errorf("inspect benchmark email")
	}
	usernameFound := usernameErr == nil
	emailFound := emailErr == nil
	if !usernameFound && !emailFound {
		return Entry{Identity: identity, Status: StatusWouldCreate}, nil, nil
	}
	if !usernameFound || !emailFound {
		return Entry{Identity: identity, Status: StatusConflict, Reason: "username/email identity is incomplete"}, nil, nil
	}
	if byUsername.ID != byEmail.ID {
		return Entry{Identity: identity, Status: StatusConflict, Reason: "username and email resolve to different users"}, nil, nil
	}
	if !isCanonical(byUsername, identity) {
		return Entry{Identity: identity, Status: StatusConflict, Reason: "existing user is not the canonical benchmark identity"}, nil, nil
	}
	return Entry{Identity: identity, Status: StatusExisting}, byUsername, nil
}

func isCanonical(user *entity.User, identity Identity) bool {
	return user.Username == identity.Username &&
		user.Email == identity.Email &&
		user.FullName == identity.FullName &&
		user.Role == rbac.RoleUser &&
		user.IsActive &&
		!user.IsSuspended
}

func (r *Result) recount() {
	r.Created, r.Skipped, r.Existing, r.Conflicts, r.Rotated = 0, 0, 0, 0, 0
	for _, entry := range r.Entries {
		switch entry.Status {
		case StatusCreated:
			r.Created++
		case StatusSkipped:
			r.Skipped++
		case StatusExisting:
			r.Existing++
		case StatusConflict:
			r.Conflicts++
		case StatusRotated:
			r.Rotated++
		}
	}
}
