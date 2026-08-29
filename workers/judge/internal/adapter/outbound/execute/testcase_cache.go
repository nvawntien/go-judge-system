package execute

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-judge-system/pkg/config"
	"go-judge-system/workers/judge/internal/application/port/outbound"

	judgepb "github.com/criyle/go-judge/pb"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	testcaseCacheNamespace  = "astracode-tc-v1"
	testcaseCacheRPCTimeout = 5 * time.Second
)

// Size is an optional suffix so existing astracode-tc-v1 entries remain
// recognizable. Newly populated entries always include it, allowing exact
// byte accounting after a Worker restart without FileGet-ing hidden inputs.
var testcaseCacheNamePattern = regexp.MustCompile(`^astracode-tc-v1-p([1-9][0-9]*)-v([1-9][0-9]*)-d([a-f0-9]{64})-t([a-f0-9]+)-i([a-f0-9]{64})(?:-s([0-9]+))?$`)

type sandboxFileRPC interface {
	FileAdd(context.Context, *judgepb.FileContent, ...grpc.CallOption) (*judgepb.FileID, error)
	FileList(context.Context, *emptypb.Empty, ...grpc.CallOption) (*judgepb.FileListType, error)
	FileDelete(context.Context, *judgepb.FileID, ...grpc.CallOption) (*emptypb.Empty, error)
}

// testcaseCacheKey is immutable provenance plus a defense-in-depth input hash.
// Its deterministic sandbox name is also the only form recognized during
// FileList reconciliation.
type testcaseCacheKey struct {
	problemID       int64
	datasetVersion  int
	datasetChecksum string
	testcaseID      string
	inputChecksum   string
}

func newTestcaseCacheKey(identity *outbound.TestcaseDatasetIdentity, testCase outbound.ExecutionTestCase) (testcaseCacheKey, bool) {
	if identity == nil || identity.ProblemID <= 0 || identity.Version <= 0 || !isSHA256Hex(identity.DatasetChecksum) || testCase.ID == "" {
		return testcaseCacheKey{}, false
	}
	inputHash := sha256.Sum256([]byte(testCase.Stdin))
	return testcaseCacheKey{
		problemID:       identity.ProblemID,
		datasetVersion:  identity.Version,
		datasetChecksum: identity.DatasetChecksum,
		testcaseID:      testCase.ID,
		inputChecksum:   hex.EncodeToString(inputHash[:]),
	}, true
}

func (k testcaseCacheKey) name() string {
	return fmt.Sprintf("%s-p%d-v%d-d%s-t%s-i%s", testcaseCacheNamespace, k.problemID, k.datasetVersion, k.datasetChecksum, hex.EncodeToString([]byte(k.testcaseID)), k.inputChecksum)
}

func (k testcaseCacheKey) nameWithSize(inputSize int64) string {
	return fmt.Sprintf("%s-s%d", k.name(), inputSize)
}

func parseTestcaseCacheName(name string) (testcaseCacheKey, int64, bool, bool) {
	matches := testcaseCacheNamePattern.FindStringSubmatch(name)
	if matches == nil {
		return testcaseCacheKey{}, 0, false, false
	}
	problemID, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil || problemID <= 0 {
		return testcaseCacheKey{}, 0, false, false
	}
	version, err := strconv.Atoi(matches[2])
	if err != nil || version <= 0 {
		return testcaseCacheKey{}, 0, false, false
	}
	testcaseIDBytes, err := hex.DecodeString(matches[4])
	if err != nil || len(testcaseIDBytes) == 0 {
		return testcaseCacheKey{}, 0, false, false
	}
	key := testcaseCacheKey{problemID: problemID, datasetVersion: version, datasetChecksum: matches[3], testcaseID: string(testcaseIDBytes), inputChecksum: matches[5]}
	if matches[6] == "" {
		return key, 0, false, true
	}
	size, err := strconv.ParseInt(matches[6], 10, 64)
	if err != nil || size < 0 {
		return testcaseCacheKey{}, 0, false, false
	}
	return key, size, true, true
}

func isSHA256Hex(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

type testcaseCacheBinding struct {
	key    testcaseCacheKey
	fileID string
}

type testcaseCacheEntry struct {
	fileID    string
	sizeBytes int64
	sizeKnown bool
	createdAt time.Time
	lastUsed  time.Time
	inUse     int
	deleting  bool
}

type sandboxTestcaseCache struct {
	client  sandboxFileRPC
	logger  *zap.Logger
	cfg     config.TestcaseCacheConfig
	enabled bool

	mu                 sync.RWMutex
	entries            map[testcaseCacheKey]*testcaseCacheEntry
	totalBytes         int64 // Exact only for entries whose names carry a size suffix.
	unknownEntries     int
	reservedBytes      int64 // In-flight FileAdd reservations; included in admission.
	reservedEntries    int
	reconciled         bool
	reconcileAttempted bool
	cleanupRunning     bool
	nextCleanup        time.Time
	loads              singleflight.Group
	list               singleflight.Group
}

func newConfiguredSandboxTestcaseCache(client sandboxFileRPC, logger *zap.Logger, cfg config.TestcaseCacheConfig) (*sandboxTestcaseCache, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	cache := &sandboxTestcaseCache{client: client, logger: logger, cfg: cfg, entries: make(map[testcaseCacheKey]*testcaseCacheEntry)}
	if !cfg.Enabled {
		return cache, nil
	}
	if cfg.MaxBytes <= 0 {
		return nil, fmt.Errorf("testcase cache max_bytes must be greater than zero when enabled")
	}
	if cfg.MaxEntries <= 0 {
		return nil, fmt.Errorf("testcase cache max_entries must be greater than zero when enabled")
	}
	if cfg.IdleTTL < 0 {
		return nil, fmt.Errorf("testcase cache idle_ttl must not be negative")
	}
	if cfg.CleanupInterval <= 0 {
		return nil, fmt.Errorf("testcase cache cleanup_interval must be greater than zero when enabled")
	}
	cache.enabled = true
	return cache, nil
}

func (c *sandboxTestcaseCache) getOrAdd(ctx context.Context, identity *outbound.TestcaseDatasetIdentity, testCase outbound.ExecutionTestCase) (testcaseCacheBinding, bool, error) {
	key, ok := newTestcaseCacheKey(identity, testCase)
	if !ok || c == nil || !c.enabled || c.client == nil {
		return testcaseCacheBinding{}, false, nil
	}
	inputSize := int64(len(testCase.Stdin))
	if err := c.reconcile(ctx); err != nil {
		c.logger.Warn("sandbox testcase cache reconciliation failed; continuing with cache population", zap.String("operation", "testcase_cache"), zap.String("event", "reconcile"), zap.Error(err))
	}
	if binding, ok := c.pin(key, inputSize); ok {
		c.logDebug("hit", key, inputSize, 0)
		return binding, true, nil
	}
	c.logDebug("miss", key, inputSize, 0)

	resultCh := c.loads.DoChan(key.name(), func() (any, error) {
		if binding, ok := c.pin(key, inputSize); ok {
			c.release([]testcaseCacheBinding{binding})
			return binding.fileID, nil
		}
		if !c.reserve(inputSize) {
			return "", errTestcaseCacheAdmissionDenied
		}
		defer c.releaseReservation(inputSize)
		rpcCtx, cancel := context.WithTimeout(context.Background(), testcaseCacheRPCTimeout)
		defer cancel()
		started := time.Now()
		fileID, err := c.client.FileAdd(rpcCtx, &judgepb.FileContent{Name: key.nameWithSize(inputSize), Content: []byte(testCase.Stdin)})
		if err != nil {
			return "", err
		}
		if fileID == nil || fileID.GetFileID() == "" {
			return "", fmt.Errorf("sandbox FileAdd returned an empty FileID")
		}
		c.store(key, fileID.GetFileID(), inputSize, true)
		c.logDebug("populate", key, inputSize, time.Since(started))
		return fileID.GetFileID(), nil
	})

	select {
	case <-ctx.Done():
		return testcaseCacheBinding{}, false, ctx.Err()
	case result := <-resultCh:
		if result.Err != nil {
			c.logger.Warn("sandbox testcase cache population failed; using MemoryFile fallback", zap.String("operation", "testcase_cache"), zap.String("event", "fallback"), zap.Int64("problem_id", key.problemID), zap.Int("dataset_version", key.datasetVersion), zap.String("dataset_checksum_prefix", checksumPrefix(key.datasetChecksum)), zap.String("testcase_id", key.testcaseID), zap.Int64("input_size_bytes", inputSize), zap.Error(result.Err))
			return testcaseCacheBinding{}, false, nil
		}
		if binding, ok := c.pin(key, inputSize); ok {
			return binding, true, nil
		}
		return testcaseCacheBinding{}, false, nil
	}
}

var errTestcaseCacheAdmissionDenied = fmt.Errorf("sandbox testcase cache admission denied")

// reserve creates an accounting-only reservation before FileAdd. It makes the
// configured byte/entry limits hard even when unrelated cache misses upload in
// parallel. We intentionally do not evict synchronously here: the sandbox
// bytes are not free until FileDelete succeeds, and a failed delete must not
// permit an unsafe upload. A denied admission falls back to MemoryFile.
func (c *sandboxTestcaseCache) reserve(size int64) bool {
	if size < 0 || size > c.cfg.MaxBytes {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// A pre-size-suffix entry was created by an older Worker and FileList does
	// not expose its size. Do not admit new bytes until it is either used (and
	// therefore accounted for) or safely evicted: treating an unknown entry as
	// zero would make MaxBytes a fiction after a Worker restart.
	if c.unknownEntries > 0 || len(c.entries)+c.reservedEntries >= c.cfg.MaxEntries || c.totalBytes+c.reservedBytes+size > c.cfg.MaxBytes {
		return false
	}
	c.reservedBytes += size
	c.reservedEntries++
	return true
}

func (c *sandboxTestcaseCache) releaseReservation(size int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reservedBytes -= size
	c.reservedEntries--
	if c.reservedBytes < 0 || c.reservedEntries < 0 {
		// This is an internal accounting invariant; clamp defensively so a
		// degraded cache cannot become an unbounded cache.
		c.reservedBytes, c.reservedEntries = 0, 0
	}
}

func (c *sandboxTestcaseCache) reconcile(ctx context.Context) error {
	if c == nil || !c.enabled || c.client == nil {
		return nil
	}
	c.mu.Lock()
	if c.reconciled || c.reconcileAttempted {
		c.mu.Unlock()
		return nil
	}
	c.reconcileAttempted = true
	c.mu.Unlock()
	resultCh := c.list.DoChan("file-list", func() (any, error) {
		rpcCtx, cancel := context.WithTimeout(context.Background(), testcaseCacheRPCTimeout)
		defer cancel()
		started := time.Now()
		response, err := c.client.FileList(rpcCtx, &emptypb.Empty{})
		if err != nil {
			return nil, err
		}
		if response == nil {
			return nil, fmt.Errorf("sandbox FileList returned nil response")
		}
		imported := 0
		// executorserver v1.7.1 returns FileListType.fileIDs as FileID ->
		// logical name. It retains names only while the sandbox process lives.
		for fileID, name := range response.GetFileIDs() {
			key, size, sizeKnown, ok := parseTestcaseCacheName(name)
			if !ok || fileID == "" {
				continue
			}
			c.store(key, fileID, size, sizeKnown)
			imported++
		}
		c.mu.Lock()
		c.reconciled = true
		c.mu.Unlock()
		c.logger.Debug("sandbox testcase cache reconciled", zap.String("operation", "testcase_cache"), zap.String("event", "reconcile"), zap.Int("entries", imported), zap.Int64("duration_ms", time.Since(started).Milliseconds()))
		return nil, nil
	})
	select {
	case <-ctx.Done():
		return ctx.Err()
	case result := <-resultCh:
		return result.Err
	}
}

// invalidateMissing verifies FileIDs through FileList rather than parsing an
// execution-error message. It returns true only when a known cached stdin file
// disappeared, permitting exactly one safe batch rebuild/retry.
func (c *sandboxTestcaseCache) invalidateMissing(ctx context.Context, bindings []testcaseCacheBinding) bool {
	if c == nil || !c.enabled || len(bindings) == 0 {
		return false
	}
	rpcCtx, cancel := context.WithTimeout(ctx, testcaseCacheRPCTimeout)
	defer cancel()
	response, err := c.client.FileList(rpcCtx, &emptypb.Empty{})
	if err != nil || response == nil {
		if err != nil {
			c.logger.Warn("sandbox testcase cache stale verification failed", zap.String("operation", "testcase_cache"), zap.String("event", "stale"), zap.Error(err))
		}
		return false
	}
	active := make(map[string]struct{}, len(response.GetFileIDs()))
	for fileID := range response.GetFileIDs() {
		active[fileID] = struct{}{}
	}
	invalidated := false
	for _, binding := range bindings {
		if _, ok := active[binding.fileID]; ok {
			continue
		}
		if c.removeIfMatch(binding.key, binding.fileID) {
			invalidated = true
			c.logDebug("stale", binding.key, 0, 0)
		}
	}
	return invalidated
}

func (c *sandboxTestcaseCache) pin(key testcaseCacheKey, inputSize int64) (testcaseCacheBinding, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || entry.deleting {
		return testcaseCacheBinding{}, false
	}
	if !entry.sizeKnown {
		entry.sizeKnown = true
		entry.sizeBytes = inputSize
		c.totalBytes += inputSize
		c.unknownEntries--
	}
	entry.lastUsed = time.Now()
	entry.inUse++
	return testcaseCacheBinding{key: key, fileID: entry.fileID}, true
}

func (c *sandboxTestcaseCache) release(bindings []testcaseCacheBinding) {
	if c == nil || !c.enabled || len(bindings) == 0 {
		return
	}
	now := time.Now()
	c.mu.Lock()
	for _, binding := range bindings {
		entry, ok := c.entries[binding.key]
		if !ok || entry.fileID != binding.fileID || entry.inUse == 0 {
			continue
		}
		entry.inUse--
		entry.lastUsed = now
	}
	c.mu.Unlock()
	c.maybeScheduleCleanup(now)
}

func (c *sandboxTestcaseCache) maybeScheduleCleanup(now time.Time) {
	if c == nil || !c.enabled {
		return
	}
	c.mu.Lock()
	if c.cleanupRunning || now.Before(c.nextCleanup) {
		c.mu.Unlock()
		return
	}
	c.cleanupRunning = true
	c.nextCleanup = now.Add(c.cfg.CleanupInterval)
	c.mu.Unlock()
	go func() {
		defer func() {
			c.mu.Lock()
			c.cleanupRunning = false
			c.mu.Unlock()
		}()
		c.cleanup(context.Background())
	}()
}

// cleanup is invoked at a bounded cadence after a batch releases its pins. It
// never holds c.mu while making a sandbox RPC.
func (c *sandboxTestcaseCache) cleanup(ctx context.Context) {
	if c == nil || !c.enabled || c.client == nil {
		return
	}
	candidates := c.selectEvictionCandidates(time.Now())
	for _, candidate := range candidates {
		rpcCtx, cancel := context.WithTimeout(ctx, testcaseCacheRPCTimeout)
		_, err := c.client.FileDelete(rpcCtx, &judgepb.FileID{FileID: candidate.fileID})
		cancel()
		if err != nil && status.Code(err) != codes.NotFound {
			c.markDeleteFailed(candidate)
			c.logger.Warn("sandbox testcase cache eviction failed", zap.String("operation", "testcase_cache"), zap.String("event", "delete_failure"), zap.Error(err))
			continue
		}
		if c.removeIfMatch(candidate.key, candidate.fileID) {
			c.logDebug("eviction", candidate.key, 0, 0)
		}
	}
}

func (c *sandboxTestcaseCache) selectEvictionCandidates(now time.Time) []testcaseCacheBinding {
	c.mu.Lock()
	defer c.mu.Unlock()
	type candidate struct {
		key   testcaseCacheKey
		entry *testcaseCacheEntry
	}
	eligible := make([]candidate, 0, len(c.entries))
	for key, entry := range c.entries {
		if entry.inUse != 0 || entry.deleting {
			continue
		}
		eligible = append(eligible, candidate{key: key, entry: entry})
	}
	sort.Slice(eligible, func(i, j int) bool {
		if eligible[i].entry.lastUsed.Equal(eligible[j].entry.lastUsed) {
			return eligible[i].key.name() < eligible[j].key.name()
		}
		return eligible[i].entry.lastUsed.Before(eligible[j].entry.lastUsed)
	})
	entries := len(c.entries)
	bytes := c.totalBytes
	bindings := make([]testcaseCacheBinding, 0)
	for _, item := range eligible {
		idle := c.cfg.IdleTTL > 0 && !item.entry.lastUsed.Add(c.cfg.IdleTTL).After(now)
		if !idle && entries <= c.cfg.MaxEntries && bytes <= c.cfg.MaxBytes {
			continue
		}
		item.entry.deleting = true
		entries--
		if item.entry.sizeKnown {
			bytes -= item.entry.sizeBytes
		}
		bindings = append(bindings, testcaseCacheBinding{key: item.key, fileID: item.entry.fileID})
	}
	return bindings
}

func (c *sandboxTestcaseCache) markDeleteFailed(binding testcaseCacheBinding) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry, ok := c.entries[binding.key]; ok && entry.fileID == binding.fileID {
		entry.deleting = false
	}
}

func (c *sandboxTestcaseCache) lookup(key testcaseCacheKey) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok || entry.deleting {
		return "", false
	}
	return entry.fileID, true
}

func (c *sandboxTestcaseCache) store(key testcaseCacheKey, fileID string, size int64, sizeKnown bool) {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if existing, ok := c.entries[key]; ok {
		if existing.fileID == fileID {
			if !existing.sizeKnown && sizeKnown {
				existing.sizeKnown, existing.sizeBytes = true, size
				c.totalBytes += size
				c.unknownEntries--
			}
			return
		}
		c.removeAccounting(existing)
	}
	entry := &testcaseCacheEntry{fileID: fileID, sizeBytes: size, sizeKnown: sizeKnown, createdAt: now, lastUsed: now}
	c.entries[key] = entry
	if sizeKnown {
		c.totalBytes += size
	} else {
		c.unknownEntries++
	}
}

func (c *sandboxTestcaseCache) removeIfMatch(key testcaseCacheKey, fileID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || entry.fileID != fileID {
		return false
	}
	c.removeAccounting(entry)
	delete(c.entries, key)
	return true
}

func (c *sandboxTestcaseCache) removeAccounting(entry *testcaseCacheEntry) {
	if entry.sizeKnown {
		c.totalBytes -= entry.sizeBytes
	} else {
		c.unknownEntries--
	}
}

func (c *sandboxTestcaseCache) logDebug(event string, key testcaseCacheKey, inputSize int64, duration time.Duration) {
	entries, bytes, unknown := c.stats()
	c.logger.Debug("sandbox testcase cache",
		zap.String("operation", "testcase_cache"),
		zap.String("event", event),
		zap.Int64("problem_id", key.problemID),
		zap.Int("dataset_version", key.datasetVersion),
		zap.String("dataset_checksum_prefix", checksumPrefix(key.datasetChecksum)),
		zap.String("testcase_id", key.testcaseID),
		zap.Int64("input_size_bytes", inputSize),
		zap.Int64("duration_ms", duration.Milliseconds()),
		zap.Int("cache_entries", entries),
		zap.Int64("cache_bytes", bytes),
		zap.Int("cache_unknown_size_entries", unknown),
	)
}

func (c *sandboxTestcaseCache) stats() (int, int64, int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries), c.totalBytes, c.unknownEntries
}

func checksumPrefix(checksum string) string {
	if len(checksum) <= 12 {
		return checksum
	}
	return checksum[:12]
}
