package problem

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go-judge-system/pkg/rbac"
	inbound "go-judge-system/services/problem/internal/application/port/inbound"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"

	"go.uber.org/zap"
)

type fakeSubmissionProblemRepository struct {
	mu       sync.Mutex
	metadata map[int64]outbound.SubmissionProblemMetadata
	errs     map[int64]error
	calls    map[int64]int
	block    <-chan struct{}
	entered  chan<- int64
}

func (r *fakeSubmissionProblemRepository) GetSubmissionProblem(ctx context.Context, id int64) (outbound.SubmissionProblemMetadata, error) {
	r.mu.Lock()
	if r.calls == nil {
		r.calls = make(map[int64]int)
	}
	r.calls[id]++
	block := r.block
	err := r.errs[id]
	metadata := r.metadata[id]
	entered := r.entered
	r.mu.Unlock()
	if entered != nil {
		entered <- id
	}

	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			return outbound.SubmissionProblemMetadata{}, ctx.Err()
		}
	}
	if err != nil {
		return outbound.SubmissionProblemMetadata{}, err
	}
	if metadata.ID == 0 {
		return outbound.SubmissionProblemMetadata{}, domain.ErrProblemNotFound
	}
	return metadata, nil
}

func (r *fakeSubmissionProblemRepository) callCount(id int64) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls[id]
}

func (r *fakeSubmissionProblemRepository) setError(id int64, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.errs == nil {
		r.errs = make(map[int64]error)
	}
	r.errs[id] = err
}

type fakeSubmissionProblemCache struct {
	mu        sync.Mutex
	entries   map[int64]outbound.SubmissionProblemMetadata
	getErr    error
	setErr    error
	deleteErr error
	getBlock  <-chan struct{}
	gets      int
	sets      int
	deletes   []int64
	lastTTL   time.Duration
}

func (c *fakeSubmissionProblemCache) Get(ctx context.Context, id int64) (outbound.SubmissionProblemMetadata, bool, error) {
	c.mu.Lock()
	c.gets++
	getBlock := c.getBlock
	getErr := c.getErr
	metadata, ok := c.entries[id]
	c.mu.Unlock()

	if getBlock != nil {
		select {
		case <-getBlock:
		case <-ctx.Done():
			return outbound.SubmissionProblemMetadata{}, false, ctx.Err()
		}
	}
	if getErr != nil {
		return outbound.SubmissionProblemMetadata{}, false, getErr
	}
	return metadata, ok, nil
}

func (c *fakeSubmissionProblemCache) Set(_ context.Context, metadata outbound.SubmissionProblemMetadata, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sets++
	c.lastTTL = ttl
	if c.setErr != nil {
		return c.setErr
	}
	if c.entries == nil {
		c.entries = make(map[int64]outbound.SubmissionProblemMetadata)
	}
	c.entries[metadata.ID] = metadata
	return nil
}

func (c *fakeSubmissionProblemCache) Delete(_ context.Context, id int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deletes = append(c.deletes, id)
	delete(c.entries, id)
	return c.deleteErr
}

func (c *fakeSubmissionProblemCache) getCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.gets
}

func cachedProblem(id int64, hidden bool, authorID string) outbound.SubmissionProblemMetadata {
	return outbound.SubmissionProblemMetadata{
		ID: id, Title: "Two Sum", Slug: "two-sum", TimeLimit: 1000, MemoryLimit: 256, IsHidden: hidden, AuthorID: authorID,
	}
}

func newCachedUseCase(repo outbound.SubmissionProblemRepository, cache outbound.SubmissionProblemCache) inbound.GetProblemUseCase {
	return NewCachedGetProblemUseCase(repo, cache, zap.NewNop())
}

func cachedRequest(problemID int64, role rbac.Role, actorID string) inbound.GetProblemRequest {
	return inbound.GetProblemRequest{ProblemID: problemID, ActorRole: role, ActorUserID: actorID}
}

func TestCachedGetProblemCacheHitSkipsDatabase(t *testing.T) {
	metadata := cachedProblem(42, false, "author")
	repo := &fakeSubmissionProblemRepository{metadata: map[int64]outbound.SubmissionProblemMetadata{42: metadata}}
	cache := &fakeSubmissionProblemCache{entries: map[int64]outbound.SubmissionProblemMetadata{42: metadata}}

	got, err := newCachedUseCase(repo, cache).Execute(context.Background(), cachedRequest(42, rbac.RoleUser, "user"))
	if err != nil || got.ID != 42 || got.Title != "Two Sum" {
		t.Fatalf("Execute() result/error = %+v/%v", got, err)
	}
	if repo.callCount(42) != 0 {
		t.Fatalf("database calls = %d, want 0", repo.callCount(42))
	}
}

func TestCachedGetProblemMissLoadsDatabaseAndPopulatesCache(t *testing.T) {
	metadata := cachedProblem(42, false, "author")
	repo := &fakeSubmissionProblemRepository{metadata: map[int64]outbound.SubmissionProblemMetadata{42: metadata}}
	cache := &fakeSubmissionProblemCache{}

	got, err := newCachedUseCase(repo, cache).Execute(context.Background(), cachedRequest(42, rbac.RoleUser, "user"))
	if err != nil || got.ID != 42 {
		t.Fatalf("Execute() result/error = %+v/%v", got, err)
	}
	if repo.callCount(42) != 1 || cache.sets != 1 || cache.lastTTL != submissionProblemCacheTTL {
		t.Fatalf("db calls/cache sets/ttl = %d/%d/%s, want 1/1/%s", repo.callCount(42), cache.sets, cache.lastTTL, submissionProblemCacheTTL)
	}
}

func TestCachedGetProblemCacheFailuresFallBackWithoutChangingBusinessResult(t *testing.T) {
	metadata := cachedProblem(42, false, "author")
	tests := []struct {
		name  string
		cache *fakeSubmissionProblemCache
	}{
		{name: "get failure", cache: &fakeSubmissionProblemCache{getErr: errors.New("redis unavailable")}},
		{name: "set failure", cache: &fakeSubmissionProblemCache{setErr: errors.New("redis unavailable")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeSubmissionProblemRepository{metadata: map[int64]outbound.SubmissionProblemMetadata{42: metadata}}
			got, err := newCachedUseCase(repo, tt.cache).Execute(context.Background(), cachedRequest(42, rbac.RoleUser, "user"))
			if err != nil || got.ID != 42 || repo.callCount(42) != 1 {
				t.Fatalf("result/error/db calls = %+v/%v/%d", got, err, repo.callCount(42))
			}
		})
	}
}

func TestCachedGetProblemBoundsCacheFailureBeforeDatabaseFallback(t *testing.T) {
	cacheBlock := make(chan struct{})
	cache := &fakeSubmissionProblemCache{getBlock: cacheBlock}
	repo := &fakeSubmissionProblemRepository{metadata: map[int64]outbound.SubmissionProblemMetadata{42: cachedProblem(42, false, "author")}}

	started := time.Now()
	got, err := newCachedUseCase(repo, cache).Execute(
		context.Background(),
		cachedRequest(42, rbac.RoleUser, "user"),
	)
	if err != nil || got.ID != 42 {
		t.Fatalf("Execute() result/error = %+v/%v", got, err)
	}
	if elapsed := time.Since(started); elapsed > 4*submissionProblemCacheTimeout {
		t.Fatalf("cache fallback took %s, want bounded cache attempts before database fallback", elapsed)
	}
	if repo.callCount(42) != 1 {
		t.Fatalf("database calls = %d, want 1", repo.callCount(42))
	}
}

func TestCachedGetProblemSharedLoadDeadlineLeavesCallerHeadroom(t *testing.T) {
	cacheBlock := make(chan struct{})
	databaseBlock := make(chan struct{})
	cache := &fakeSubmissionProblemCache{getBlock: cacheBlock}
	repo := &fakeSubmissionProblemRepository{
		metadata: map[int64]outbound.SubmissionProblemMetadata{42: cachedProblem(42, false, "author")},
		block:    databaseBlock,
	}
	callerCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	started := time.Now()
	_, err := newCachedUseCase(repo, cache).Execute(callerCtx, cachedRequest(42, rbac.RoleUser, "user"))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Execute() error = %v, want bounded shared-load deadline", err)
	}
	if elapsed := time.Since(started); elapsed > 900*time.Millisecond {
		t.Fatalf("shared load consumed %s of the one-second caller budget", elapsed)
	}
}

func TestCachedGetProblemDoesNotCacheDatabaseFailures(t *testing.T) {
	cache := &fakeSubmissionProblemCache{}
	repo := &fakeSubmissionProblemRepository{errs: map[int64]error{42: errors.New("database unavailable")}}
	_, err := newCachedUseCase(repo, cache).Execute(context.Background(), cachedRequest(42, rbac.RoleUser, "user"))
	if !errors.Is(err, domain.ErrInternalServer) {
		t.Fatalf("Execute() error = %v, want internal server", err)
	}
	if cache.sets != 0 {
		t.Fatalf("cache writes = %d, want 0", cache.sets)
	}
}

func TestCachedGetProblemRetriesAfterFailedSharedLoad(t *testing.T) {
	cache := &fakeSubmissionProblemCache{}
	repo := &fakeSubmissionProblemRepository{metadata: map[int64]outbound.SubmissionProblemMetadata{42: cachedProblem(42, false, "author")}}
	repo.setError(42, errors.New("database unavailable"))
	useCase := newCachedUseCase(repo, cache)

	if _, err := useCase.Execute(context.Background(), cachedRequest(42, rbac.RoleUser, "user")); !errors.Is(err, domain.ErrInternalServer) {
		t.Fatalf("first Execute() error = %v, want internal server", err)
	}
	repo.setError(42, nil)
	if got, err := useCase.Execute(context.Background(), cachedRequest(42, rbac.RoleUser, "user")); err != nil || got.ID != 42 {
		t.Fatalf("second Execute() result/error = %+v/%v", got, err)
	}
	if repo.callCount(42) != 2 {
		t.Fatalf("database calls = %d, want failed load plus retry", repo.callCount(42))
	}
}

func TestCachedGetProblemPreservesHiddenAccessMatrix(t *testing.T) {
	metadata := cachedProblem(42, true, "owner")
	repo := &fakeSubmissionProblemRepository{}
	cache := &fakeSubmissionProblemCache{entries: map[int64]outbound.SubmissionProblemMetadata{42: metadata}}
	useCase := newCachedUseCase(repo, cache)
	tests := []struct {
		name  string
		role  rbac.Role
		actor string
		allow bool
	}{
		{name: "normal user", role: rbac.RoleUser, actor: "user"},
		{name: "owner contributor", role: rbac.RoleContributor, actor: "owner", allow: true},
		{name: "other contributor", role: rbac.RoleContributor, actor: "other"},
		{name: "moderator", role: rbac.RoleModerator, actor: "moderator", allow: true},
		{name: "admin", role: rbac.RoleAdmin, actor: "admin", allow: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := useCase.Execute(context.Background(), cachedRequest(42, tt.role, tt.actor))
			if tt.allow && err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if !tt.allow && !errors.Is(err, domain.ErrProblemNotFound) {
				t.Fatalf("Execute() error = %v, want not found", err)
			}
		})
	}
	if repo.callCount(42) != 0 {
		t.Fatalf("database calls = %d, want cache-only authorization", repo.callCount(42))
	}
}

func TestCachedGetProblemNotFound(t *testing.T) {
	cache := &fakeSubmissionProblemCache{}
	repo := &fakeSubmissionProblemRepository{metadata: map[int64]outbound.SubmissionProblemMetadata{}}
	_, err := newCachedUseCase(repo, cache).Execute(context.Background(), cachedRequest(42, rbac.RoleUser, "user"))
	if !errors.Is(err, domain.ErrProblemNotFound) {
		t.Fatalf("Execute() error = %v, want not found", err)
	}
	if cache.sets != 0 {
		t.Fatalf("cache writes = %d, want no negative cache entry", cache.sets)
	}
}

func TestCachedGetProblemSingleflightCoalescesColdCache(t *testing.T) {
	const callers = 1000
	metadata := cachedProblem(42, false, "author")
	release := make(chan struct{})
	repo := &fakeSubmissionProblemRepository{
		metadata: map[int64]outbound.SubmissionProblemMetadata{42: metadata},
		block:    release,
	}
	cache := &fakeSubmissionProblemCache{}
	useCase := newCachedUseCase(repo, cache)

	var wg sync.WaitGroup
	errs := make(chan error, callers)
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := useCase.Execute(context.Background(), cachedRequest(42, rbac.RoleUser, "user"))
			if err == nil && got.ID != 42 {
				err = errors.New("unexpected metadata")
			}
			errs <- err
		}()
	}

	deadline := time.Now().Add(time.Second)
	for cache.getCount() < callers && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if cache.getCount() < callers {
		t.Fatalf("cache reads = %d, want at least %d concurrent cold-cache callers", cache.getCount(), callers)
	}
	if repo.callCount(42) != 1 {
		t.Fatalf("database calls while cold load is blocked = %d, want exactly 1", repo.callCount(42))
	}
	close(release)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Execute() error = %v", err)
		}
	}
	if repo.callCount(42) != 1 {
		t.Fatalf("database calls = %d, want exactly 1", repo.callCount(42))
	}
}

func TestCachedGetProblemWarmCacheAvoidsDatabaseUnderConcurrency(t *testing.T) {
	const callers = 1000
	metadata := cachedProblem(42, false, "author")
	repo := &fakeSubmissionProblemRepository{}
	cache := &fakeSubmissionProblemCache{entries: map[int64]outbound.SubmissionProblemMetadata{42: metadata}}
	useCase := newCachedUseCase(repo, cache)

	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := useCase.Execute(context.Background(), cachedRequest(42, rbac.RoleUser, "user")); err != nil {
				t.Errorf("Execute() error = %v", err)
			}
		}()
	}
	wg.Wait()
	if repo.callCount(42) != 0 {
		t.Fatalf("database calls = %d, want 0", repo.callCount(42))
	}
}

func TestCachedGetProblemSingleflightDoesNotShareDifferentProblemIDs(t *testing.T) {
	release := make(chan struct{})
	entered := make(chan int64, 2)
	repo := &fakeSubmissionProblemRepository{metadata: map[int64]outbound.SubmissionProblemMetadata{
		42: cachedProblem(42, false, "author"),
		43: cachedProblem(43, false, "author"),
	}, block: release, entered: entered}
	cache := &fakeSubmissionProblemCache{}
	useCase := newCachedUseCase(repo, cache)

	var wg sync.WaitGroup
	for _, id := range []int64{42, 43} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := useCase.Execute(context.Background(), cachedRequest(id, rbac.RoleUser, "user")); err != nil {
				t.Errorf("Execute() error = %v", err)
			}
		}()
	}

	seen := map[int64]bool{}
	for range 2 {
		select {
		case id := <-entered:
			seen[id] = true
		case <-time.After(time.Second):
			t.Fatalf("different problem IDs did not reach independent database loaders; seen=%v", seen)
		}
	}
	if !seen[42] || !seen[43] {
		t.Fatalf("database loader IDs = %v, want both 42 and 43", seen)
	}
	close(release)
	wg.Wait()
	if repo.callCount(42) != 1 || repo.callCount(43) != 1 {
		t.Fatalf("database calls = %d/%d, want 1/1", repo.callCount(42), repo.callCount(43))
	}
}

func TestCachedGetProblemFollowerCancellationDoesNotCancelSharedLoad(t *testing.T) {
	metadata := cachedProblem(42, false, "author")
	release := make(chan struct{})
	repo := &fakeSubmissionProblemRepository{metadata: map[int64]outbound.SubmissionProblemMetadata{42: metadata}, block: release}
	cache := &fakeSubmissionProblemCache{}
	useCase := newCachedUseCase(repo, cache)

	leaderResult := make(chan error, 1)
	go func() {
		_, err := useCase.Execute(context.Background(), cachedRequest(42, rbac.RoleUser, "leader"))
		leaderResult <- err
	}()

	deadline := time.Now().Add(time.Second)
	for repo.callCount(42) == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if repo.callCount(42) != 1 {
		t.Fatalf("database calls before follower joins = %d, want 1", repo.callCount(42))
	}

	followerCtx, cancel := context.WithCancel(context.Background())
	followerResult := make(chan error, 1)
	go func() {
		_, err := useCase.Execute(followerCtx, cachedRequest(42, rbac.RoleUser, "follower"))
		followerResult <- err
	}()
	deadline = time.Now().Add(time.Second)
	for cache.getCount() < 3 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if cache.getCount() < 3 {
		t.Fatalf("cache reads = %d, follower did not join the shared load", cache.getCount())
	}
	cancel()
	if err := <-followerResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("follower error = %v, want context canceled", err)
	}

	close(release)
	if err := <-leaderResult; err != nil {
		t.Fatalf("leader error = %v", err)
	}
	if repo.callCount(42) != 1 {
		t.Fatalf("database calls = %d, want 1", repo.callCount(42))
	}
}

func TestCachedGetProblemCallerCancellationDoesNotCancelSharedLoad(t *testing.T) {
	metadata := cachedProblem(42, false, "author")
	release := make(chan struct{})
	repo := &fakeSubmissionProblemRepository{metadata: map[int64]outbound.SubmissionProblemMetadata{42: metadata}, block: release}
	cache := &fakeSubmissionProblemCache{}
	useCase := newCachedUseCase(repo, cache)

	canceledCtx, cancel := context.WithCancel(context.Background())
	canceledResult := make(chan error, 1)
	go func() {
		_, err := useCase.Execute(canceledCtx, cachedRequest(42, rbac.RoleUser, "user"))
		canceledResult <- err
	}()

	deadline := time.Now().Add(time.Second)
	for repo.callCount(42) == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	cancel()
	if err := <-canceledResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled caller error = %v, want context canceled", err)
	}

	completed := make(chan error, 1)
	go func() {
		_, err := useCase.Execute(context.Background(), cachedRequest(42, rbac.RoleUser, "user"))
		completed <- err
	}()
	close(release)
	if err := <-completed; err != nil {
		t.Fatalf("shared caller error = %v", err)
	}
	if repo.callCount(42) != 1 {
		t.Fatalf("database calls = %d, want 1", repo.callCount(42))
	}
}
