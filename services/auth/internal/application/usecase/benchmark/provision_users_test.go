package benchmark

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go-judge-system/pkg/rbac"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"
)

func TestIdentities(t *testing.T) {
	identities, err := Identities(1, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct {
		index int
		user  string
		mail  string
		name  string
	}{
		{0, "benchmark_judge_001", "benchmark-judge-001@benchmark.invalid", "Benchmark Judge 001"},
		{49, "benchmark_judge_050", "benchmark-judge-050@benchmark.invalid", "Benchmark Judge 050"},
		{99, "benchmark_judge_100", "benchmark-judge-100@benchmark.invalid", "Benchmark Judge 100"},
	} {
		got := identities[want.index]
		if got.Username != want.user || got.Email != want.mail || got.FullName != want.name {
			t.Fatalf("identity[%d]=%+v", want.index, got)
		}
	}
}

func TestIdentitiesRejectInvalidRanges(t *testing.T) {
	for _, input := range [][2]int{{0, 1}, {1, 0}, {101, 1}, {100, 2}, {1, 101}, {1, int(^uint(0) >> 1)}} {
		if _, err := Identities(input[0], input[1]); !errors.Is(err, ErrInvalidRange) {
			t.Fatalf("Identities(%d,%d) error=%v", input[0], input[1], err)
		}
	}
}

func TestDryRunDoesNotUsePasswordOrWrite(t *testing.T) {
	repo := newFakeRepo()
	encoder := &fakeEncoder{}
	result, err := NewProvisioner(repo, encoder).Execute(context.Background(), Request{Start: 1, Count: 2})
	if err != nil {
		t.Fatal(err)
	}
	if result.Entries[0].Status != StatusWouldCreate || repo.creates != 0 || encoder.hashes != 0 || encoder.compares != 0 {
		t.Fatalf("result=%+v creates=%d hashes=%d compares=%d", result, repo.creates, encoder.hashes, encoder.compares)
	}
}

func TestApplyCreatesActiveNormalUsersAndRerunSkips(t *testing.T) {
	repo := newFakeRepo()
	encoder := &fakeEncoder{expected: "correct-password"}
	provisioner := NewProvisioner(repo, encoder)
	result, err := provisioner.Execute(context.Background(), Request{Start: 1, Count: 2, Apply: true, Password: []byte("correct-password")})
	if err != nil || result.Created != 2 || repo.creates != 2 || encoder.hashes != 2 {
		t.Fatalf("result=%+v err=%v creates=%d hashes=%d", result, err, repo.creates, encoder.hashes)
	}
	for _, user := range repo.usersByUsername {
		if user.Role != rbac.RoleUser || !user.IsActive || user.IsSuspended || user.Rating != 0 || user.Password == "correct-password" {
			t.Fatalf("unexpected created user: %+v", user)
		}
	}
	result, err = provisioner.Execute(context.Background(), Request{Start: 1, Count: 2, Apply: true, Password: []byte("correct-password")})
	if err != nil || result.Skipped != 2 || repo.creates != 2 {
		t.Fatalf("rerun result=%+v err=%v creates=%d", result, err, repo.creates)
	}
}

func TestApplyVisibleConflictAbortsBeforeCreates(t *testing.T) {
	repo := newFakeRepo()
	repo.usersByUsername["benchmark_judge_002"] = canonicalUser(2, "wrong-hash")
	// Deliberately leave the matching email absent: this is a username-only collision.
	encoder := &fakeEncoder{expected: "password-1"}
	result, err := NewProvisioner(repo, encoder).Execute(context.Background(), Request{Start: 1, Count: 2, Apply: true, Password: []byte("password-1")})
	if !errors.Is(err, ErrConflicts) || result.Conflicts != 1 || repo.creates != 0 {
		t.Fatalf("result=%+v err=%v creates=%d", result, err, repo.creates)
	}
}

func TestPhaseAConflictAtAccount050PreventsAllCreates(t *testing.T) {
	repo := newFakeRepo()
	repo.usersByUsername["benchmark_judge_050"] = canonicalUser(50, "password")
	result, err := NewProvisioner(repo, &fakeEncoder{expected: "password"}).Execute(context.Background(), Request{Start: 1, Count: 50, Apply: true, Password: []byte("password")})
	if !errors.Is(err, ErrConflicts) || result.Conflicts != 1 || repo.creates != 0 {
		t.Fatalf("result=%+v err=%v creates=%d", result, err, repo.creates)
	}
}

func TestApplyCanonicalPasswordMismatchIsConflict(t *testing.T) {
	repo := newFakeRepo()
	user := canonicalUser(1, "other-password")
	repo.put(user)
	encoder := &fakeEncoder{expected: "expected-password"}
	result, err := NewProvisioner(repo, encoder).Execute(context.Background(), Request{Start: 1, Count: 1, Apply: true, Password: []byte("expected-password")})
	if !errors.Is(err, ErrConflicts) || result.Conflicts != 1 || repo.creates != 0 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCanonicalityConflictsNeverMutateExistingUsers(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*fakeRepo, *entity.User)
	}{
		{"email-only", func(repo *fakeRepo, user *entity.User) { repo.usersByEmail[user.Email] = user }},
		{"split-identities", func(repo *fakeRepo, user *entity.User) {
			repo.usersByUsername[user.Username] = user
			repo.usersByEmail[user.Email] = canonicalUser(2, "password")
		}},
		{"wrong-full-name", func(repo *fakeRepo, user *entity.User) { user.FullName = "Not Benchmark"; repo.put(user) }},
		{"wrong-role", func(repo *fakeRepo, user *entity.User) { user.Role = rbac.RoleAdmin; repo.put(user) }},
		{"inactive", func(repo *fakeRepo, user *entity.User) { user.IsActive = false; repo.put(user) }},
		{"suspended", func(repo *fakeRepo, user *entity.User) { user.IsSuspended = true; repo.put(user) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := newFakeRepo()
			user := canonicalUser(1, "password")
			test.prepare(repo, user)
			result, err := NewProvisioner(repo, &fakeEncoder{expected: "password"}).Execute(context.Background(), Request{Start: 1, Count: 1, Apply: true, Password: []byte("password")})
			if !errors.Is(err, ErrConflicts) || result.Conflicts != 1 || repo.creates != 0 {
				t.Fatalf("result=%+v err=%v creates=%d", result, err, repo.creates)
			}
		})
	}
}

func TestCreateRaceCanonicalMatchingPasswordSkips(t *testing.T) {
	repo := newFakeRepo()
	repo.createErr = errors.New("duplicate")
	repo.onCreateError = func(user *entity.User) { repo.put(canonicalUser(1, "password")) }
	encoder := &fakeEncoder{expected: "password"}
	result, err := NewProvisioner(repo, encoder).Execute(context.Background(), Request{Start: 1, Count: 1, Apply: true, Password: []byte("password")})
	if err != nil || result.Skipped != 1 || repo.creates != 1 {
		t.Fatalf("result=%+v err=%v creates=%d", result, err, repo.creates)
	}
}

func TestCreateFailureWithoutCanonicalRereadStopsSafely(t *testing.T) {
	repo := newFakeRepo()
	repo.createErr = errors.New("database unavailable")
	result, err := NewProvisioner(repo, &fakeEncoder{expected: "password"}).Execute(context.Background(), Request{Start: 1, Count: 1, Apply: true, Password: []byte("password")})
	if !errors.Is(err, ErrApplyStopped) || result.Skipped != 0 || result.Created != 0 || result.StoppedAt == nil || result.StoppedAt.Sequence != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestPartialFailureStopsLaterUsersAndRerunRecovers(t *testing.T) {
	repo := newFakeRepo()
	repo.failCreateAt = 2
	encoder := &fakeEncoder{expected: "password"}
	provisioner := NewProvisioner(repo, encoder)
	result, err := provisioner.Execute(context.Background(), Request{Start: 1, Count: 3, Apply: true, Password: []byte("password")})
	if !errors.Is(err, ErrApplyStopped) || result.Created != 1 || repo.creates != 2 || result.StoppedAt == nil || result.StoppedAt.Sequence != 2 {
		t.Fatalf("result=%+v err=%v creates=%d", result, err, repo.creates)
	}
	delete(repo.usersByUsername, "benchmark_judge_002")
	delete(repo.usersByEmail, "benchmark-judge-002@benchmark.invalid")
	repo.failCreateAt = 0
	result, err = provisioner.Execute(context.Background(), Request{Start: 1, Count: 3, Apply: true, Password: []byte("password")})
	if err != nil || result.Skipped != 1 || result.Created != 2 {
		t.Fatalf("rerun result=%+v err=%v", result, err)
	}
}

func TestCancellationPreventsCreates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	repo := newFakeRepo()
	result, err := NewProvisioner(repo, &fakeEncoder{expected: "password"}).Execute(ctx, Request{Start: 1, Count: 1, Apply: true, Password: []byte("password")})
	if !errors.Is(err, context.Canceled) || repo.creates != 0 || len(result.Entries) != 0 {
		t.Fatalf("result=%+v err=%v creates=%d", result, err, repo.creates)
	}
}

type fakeEncoder struct {
	expected string
	hashes   int
	compares int
}

func (f *fakeEncoder) HashAndSalt(password []byte) (string, error) {
	f.hashes++
	return "hash:" + string(password), nil
}

func (f *fakeEncoder) ComparePasswords(hash string, plain []byte) bool {
	f.compares++
	return hash == "hash:"+string(plain) || string(plain) == f.expected && hash == f.expected
}

type fakeRepo struct {
	usersByUsername map[string]*entity.User
	usersByEmail    map[string]*entity.User
	creates         int
	failCreateAt    int
	createErr       error
	onCreateError   func(*entity.User)
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{usersByUsername: map[string]*entity.User{}, usersByEmail: map[string]*entity.User{}}
}

func (r *fakeRepo) put(user *entity.User) {
	r.usersByUsername[user.Username] = user
	r.usersByEmail[user.Email] = user
}

func (r *fakeRepo) CreateUser(_ context.Context, user *entity.User) error {
	r.creates++
	if r.createErr != nil || r.failCreateAt == r.creates {
		if r.onCreateError != nil {
			r.onCreateError(user)
		}
		if r.createErr != nil {
			return r.createErr
		}
		return errors.New("create failed")
	}
	r.put(user)
	return nil
}
func (r *fakeRepo) GetUserByEmail(_ context.Context, email string) (*entity.User, error) {
	if user, ok := r.usersByEmail[email]; ok {
		return user, nil
	}
	return nil, domain.ErrUserNotFound
}
func (r *fakeRepo) GetUserByUsername(_ context.Context, username string) (*entity.User, error) {
	if user, ok := r.usersByUsername[username]; ok {
		return user, nil
	}
	return nil, domain.ErrUserNotFound
}
func (r *fakeRepo) GetUserById(context.Context, string) (*entity.User, error) {
	return nil, domain.ErrUserNotFound
}
func (r *fakeRepo) ListUsers(context.Context, outbound.ListUsersFilter) (outbound.ListUsersResult, error) {
	return outbound.ListUsersResult{}, nil
}
func (r *fakeRepo) SearchPublicUsers(context.Context, outbound.SearchPublicUsersFilter) (outbound.SearchPublicUsersResult, error) {
	return outbound.SearchPublicUsersResult{}, nil
}
func (r *fakeRepo) UpdateUser(context.Context, *entity.User) error {
	return errors.New("unexpected update")
}
func (r *fakeRepo) UpdatePassword(context.Context, string, string, time.Time) error {
	return errors.New("unexpected password update")
}
func (r *fakeRepo) UpdateProfile(context.Context, string, outbound.ProfileUpdates) error {
	return errors.New("unexpected profile update")
}
func (r *fakeRepo) UpdateAvatar(context.Context, string, string, string, time.Time) error {
	return errors.New("unexpected avatar update")
}
func (r *fakeRepo) DeleteUser(context.Context, string) error { return errors.New("unexpected delete") }

func canonicalUser(sequence int, password string) *entity.User {
	identity, _ := Identities(sequence, 1)
	return &entity.User{ID: fmt.Sprintf("id-%03d", sequence), Username: identity[0].Username, Email: identity[0].Email, FullName: identity[0].FullName, Password: password, Role: rbac.RoleUser, IsActive: true}
}
