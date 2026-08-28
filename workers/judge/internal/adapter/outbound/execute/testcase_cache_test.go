package execute

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-judge-system/pkg/config"
	"go-judge-system/workers/judge/internal/application/port/outbound"

	judgepb "github.com/criyle/go-judge/pb"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGoJudgeClientOfficialTestcaseCacheColdThenWarm(t *testing.T) {
	identity := testDatasetIdentity(3, 2, "a")
	testCase := cachedOfficialTestCase(1, "case-1", "input\n", "ok\n")
	var fileAdds atomic.Int32
	var runCalls atomic.Int32
	rpc := &fakeExecutorRPC{
		fileList: func(context.Context, *emptypb.Empty) (*judgepb.FileListType, error) {
			return &judgepb.FileListType{}, nil
		},
		fileAdd: func(_ context.Context, file *judgepb.FileContent) (*judgepb.FileID, error) {
			fileAdds.Add(1)
			if !strings.HasPrefix(file.GetName(), testcaseCacheNamespace+"-p3-v2-") {
				t.Fatalf("FileAdd logical name = %q", file.GetName())
			}
			if got := string(file.GetContent()); got != testCase.Stdin {
				t.Fatalf("FileAdd content = %q, want testcase input", got)
			}
			return &judgepb.FileID{FileID: "stdin-cache-1"}, nil
		},
		handler: func(_ context.Context, request *judgepb.Request) (*judgepb.Response, error) {
			if isCompileRequest(request) {
				return compileAccepted("exe-1"), nil
			}
			runCalls.Add(1)
			assertCachedOfficialStdin(t, request, "stdin-cache-1")
			return acceptedResults("ok\n"), nil
		},
	}
	client := NewGoJudgeClient(rpc, zap.NewNop(), enabledTestcaseCacheConfig())
	for iteration := 0; iteration < 2; iteration++ {
		result, err := client.Execute(context.Background(), outbound.ExecutionRequest{
			Language: "CPP", SourceCode: "int main(){}", StopOnFirstFailure: true,
			TestcaseDataset: identity, TestCases: []outbound.ExecutionTestCase{testCase},
		})
		if err != nil || result.Status != "ACCEPTED" {
			t.Fatalf("iteration %d result/error = %#v/%v", iteration, result, err)
		}
	}
	if got := fileAdds.Load(); got != 1 {
		t.Fatalf("FileAdd calls = %d, want one cold-cache upload", got)
	}
	if got := runCalls.Load(); got != 2 {
		t.Fatalf("run calls = %d, want two executions", got)
	}
}

func TestSandboxTestcaseCacheIdentityIncludesDatasetAndInput(t *testing.T) {
	cache := newSandboxTestcaseCache(&fakeExecutorRPC{fileList: emptyFileList, fileAdd: sequentialFileAdd()}, zap.NewNop())
	base := cachedOfficialTestCase(1, "case-1", "input-a\n", "ok\n")
	cases := []struct {
		name     string
		identity *outbound.TestcaseDatasetIdentity
		testCase outbound.ExecutionTestCase
	}{
		{"base", testDatasetIdentity(3, 2, "a"), base},
		{"different version", testDatasetIdentity(3, 3, "b"), base},
		{"different dataset checksum", testDatasetIdentity(3, 2, "c"), base},
		{"different input hash", testDatasetIdentity(3, 2, "a"), cachedOfficialTestCase(1, "case-1", "input-b\n", "ok\n")},
		{"different testcase identity", testDatasetIdentity(3, 2, "a"), cachedOfficialTestCase(2, "case-2", "input-a\n", "ok\n")},
	}
	seen := map[string]struct{}{}
	for _, tc := range cases {
		binding, cached, err := cache.getOrAdd(context.Background(), tc.identity, tc.testCase)
		if err != nil || !cached || binding.fileID == "" {
			t.Fatalf("%s getOrAdd = %#v/%t/%v", tc.name, binding, cached, err)
		}
		if _, duplicate := seen[binding.fileID]; duplicate {
			t.Fatalf("%s reused FileID %q across a distinct cache identity", tc.name, binding.fileID)
		}
		seen[binding.fileID] = struct{}{}
	}
}

func TestSandboxTestcaseCacheConcurrentColdMissesCoalescePerKey(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var addCalls atomic.Int32
	cache := newSandboxTestcaseCache(&fakeExecutorRPC{
		fileList: emptyFileList,
		fileAdd: func(ctx context.Context, _ *judgepb.FileContent) (*judgepb.FileID, error) {
			if addCalls.Add(1) == 1 {
				close(started)
			}
			select {
			case <-release:
				return &judgepb.FileID{FileID: "shared-file"}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}, zap.NewNop())
	identity := testDatasetIdentity(3, 2, "a")
	testCase := cachedOfficialTestCase(1, "case-1", "input\n", "ok\n")

	const callers = 32
	var wg sync.WaitGroup
	results := make(chan testcaseCacheBinding, callers)
	errorsCh := make(chan error, callers)
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			binding, cached, err := cache.getOrAdd(context.Background(), identity, testCase)
			if err != nil || !cached {
				errorsCh <- fmt.Errorf("cached=%t err=%w", cached, err)
				return
			}
			results <- binding
		}()
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("FileAdd did not begin")
	}
	close(release)
	wg.Wait()
	close(results)
	close(errorsCh)
	for err := range errorsCh {
		t.Fatal(err)
	}
	if got := addCalls.Load(); got != 1 {
		t.Fatalf("FileAdd calls = %d, want one", got)
	}
	for binding := range results {
		if binding.fileID != "shared-file" {
			t.Fatalf("FileID = %q, want shared-file", binding.fileID)
		}
	}
}

func TestSandboxTestcaseCacheDifferentKeysDoNotSerializePopulation(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var addCalls atomic.Int32
	cache := newSandboxTestcaseCache(&fakeExecutorRPC{
		fileList: emptyFileList,
		fileAdd: func(ctx context.Context, _ *judgepb.FileContent) (*judgepb.FileID, error) {
			call := addCalls.Add(1)
			started <- struct{}{}
			select {
			case <-release:
				return &judgepb.FileID{FileID: fmt.Sprintf("file-%d", call)}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}, zap.NewNop())
	identity := testDatasetIdentity(3, 2, "a")
	var wg sync.WaitGroup
	for _, testCase := range []outbound.ExecutionTestCase{
		cachedOfficialTestCase(1, "case-1", "one\n", "ok\n"),
		cachedOfficialTestCase(2, "case-2", "two\n", "ok\n"),
	} {
		wg.Add(1)
		go func(testCase outbound.ExecutionTestCase) {
			defer wg.Done()
			_, _, _ = cache.getOrAdd(context.Background(), identity, testCase)
		}(testCase)
	}
	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("different testcase cache keys were serialized")
		}
	}
	close(release)
	wg.Wait()
	if got := addCalls.Load(); got != 2 {
		t.Fatalf("FileAdd calls = %d, want two independent keys", got)
	}
}

func TestGoJudgeClientTestcaseCacheFileAddFailureFallsBackToMemory(t *testing.T) {
	testCase := cachedOfficialTestCase(1, "case-1", "input\n", "ok\n")
	rpc := &fakeExecutorRPC{
		fileList: emptyFileList,
		fileAdd: func(context.Context, *judgepb.FileContent) (*judgepb.FileID, error) {
			return nil, errors.New("sandbox file store unavailable")
		},
		handler: func(_ context.Context, request *judgepb.Request) (*judgepb.Response, error) {
			if isCompileRequest(request) {
				return compileAccepted("exe"), nil
			}
			if got := string(request.GetCmd()[0].GetFiles()[0].GetMemory().GetContent()); got != testCase.Stdin {
				t.Fatalf("fallback stdin = %q, want MemoryFile input", got)
			}
			return acceptedResults("ok\n"), nil
		},
	}
	result, err := NewGoJudgeClient(rpc, zap.NewNop(), enabledTestcaseCacheConfig()).Execute(context.Background(), outbound.ExecutionRequest{
		Language: "CPP", SourceCode: "int main(){}", StopOnFirstFailure: true, TestcaseDataset: testDatasetIdentity(3, 2, "a"), TestCases: []outbound.ExecutionTestCase{testCase},
	})
	if err != nil || result.Status != "ACCEPTED" {
		t.Fatalf("result/error = %#v/%v", result, err)
	}
}

func TestSandboxTestcaseCacheReconcilesOnlyAstraCodeNames(t *testing.T) {
	identity := testDatasetIdentity(3, 2, "a")
	testCase := cachedOfficialTestCase(1, "case-1", "input\n", "ok\n")
	key, ok := newTestcaseCacheKey(identity, testCase)
	if !ok {
		t.Fatal("build key")
	}
	cache := newSandboxTestcaseCache(&fakeExecutorRPC{fileList: func(context.Context, *emptypb.Empty) (*judgepb.FileListType, error) {
		return &judgepb.FileListType{FileIDs: map[string]string{"known": key.name(), "compile-file": "main", "foreign": "astracode-tc-v1-malformed"}}, nil
	}}, zap.NewNop())
	if err := cache.reconcile(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if got, ok := cache.lookup(key); !ok || got != "known" {
		t.Fatalf("reconciled FileID = %q/%t", got, ok)
	}
	if got := len(cache.entries); got != 1 {
		t.Fatalf("reconciled entries = %d, want only recognized testcase cache entry", got)
	}
}

func TestGoJudgeClientStaleTestcaseFileIDRetriesBatchOnce(t *testing.T) {
	identity := testDatasetIdentity(3, 2, "a")
	testCase := cachedOfficialTestCase(1, "case-1", "input\n", "ok\n")
	key, _ := newTestcaseCacheKey(identity, testCase)
	var fileAdds atomic.Int32
	var runCalls atomic.Int32
	rpc := &fakeExecutorRPC{
		fileList: func(context.Context, *emptypb.Empty) (*judgepb.FileListType, error) {
			return &judgepb.FileListType{}, nil
		},
		fileAdd: func(context.Context, *judgepb.FileContent) (*judgepb.FileID, error) {
			fileAdds.Add(1)
			return &judgepb.FileID{FileID: "new-file"}, nil
		},
		handler: func(_ context.Context, request *judgepb.Request) (*judgepb.Response, error) {
			if isCompileRequest(request) {
				return compileAccepted("exe"), nil
			}
			if runCalls.Add(1) == 1 {
				assertCachedOfficialStdin(t, request, "stale-file")
				return &judgepb.Response{Results: []*judgepb.Response_Result{{Status: judgepb.Response_Result_FileError, FileError: []*judgepb.Response_FileError{{Name: "stdin", Type: judgepb.Response_FileError_CopyInOpenFile}}}}}, nil
			}
			assertCachedOfficialStdin(t, request, "new-file")
			return acceptedResults("ok\n"), nil
		},
	}
	client := NewGoJudgeClient(rpc, zap.NewNop(), enabledTestcaseCacheConfig())
	client.testcaseCache.store(key, "stale-file", 0, false)
	client.testcaseCache.reconciled = true
	result, err := client.Execute(context.Background(), outbound.ExecutionRequest{Language: "CPP", SourceCode: "int main(){}", StopOnFirstFailure: true, TestcaseDataset: identity, TestCases: []outbound.ExecutionTestCase{testCase}})
	if err != nil || result.Status != "ACCEPTED" {
		t.Fatalf("result/error = %#v/%v", result, err)
	}
	if got := fileAdds.Load(); got != 1 {
		t.Fatalf("stale recovery FileAdd calls = %d, want one", got)
	}
	if got := runCalls.Load(); got != 2 {
		t.Fatalf("stale recovery run calls = %d, want exactly one retry", got)
	}
}

func TestGoJudgeClientOfficialEarlyStopDoesNotPopulateLaterInputs(t *testing.T) {
	var fileAdds atomic.Int32
	rpc := &fakeExecutorRPC{
		fileList: emptyFileList,
		fileAdd: func(_ context.Context, _ *judgepb.FileContent) (*judgepb.FileID, error) {
			return &judgepb.FileID{FileID: fmt.Sprintf("input-%d", fileAdds.Add(1))}, nil
		},
		handler: func(_ context.Context, request *judgepb.Request) (*judgepb.Response, error) {
			if isCompileRequest(request) {
				return compileAccepted("exe"), nil
			}
			response := acceptedResults("1\n", "wrong\n", "3\n", "4\n")
			return response, nil
		},
	}
	result, err := NewGoJudgeClient(rpc, zap.NewNop(), enabledTestcaseCacheConfig()).Execute(context.Background(), outbound.ExecutionRequest{
		Language: "CPP", SourceCode: "int main(){}", StopOnFirstFailure: true, TestcaseDataset: testDatasetIdentity(3, 2, "a"), TestCases: officialTestCases(8),
	})
	if err != nil || result.Status != "WRONG_ANSWER" || len(result.TestCases) != 2 {
		t.Fatalf("result/error = %#v/%v", result, err)
	}
	if got := fileAdds.Load(); got != officialBatchSize {
		t.Fatalf("FileAdd calls = %d, want only first K=%d testcase inputs", got, officialBatchSize)
	}
}

func TestSandboxTestcaseCacheCallerCancellationReturnsPromptly(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	cache := newSandboxTestcaseCache(&fakeExecutorRPC{
		fileList: emptyFileList,
		fileAdd: func(ctx context.Context, _ *judgepb.FileContent) (*judgepb.FileID, error) {
			close(started)
			select {
			case <-release:
				return &judgepb.FileID{FileID: "shared-file"}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}, zap.NewNop())
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, _, err := cache.getOrAdd(ctx, testDatasetIdentity(3, 2, "a"), cachedOfficialTestCase(1, "case-1", "input\n", "ok\n"))
		result <- err
	}()
	<-started
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("caller did not return promptly after cancellation")
	}
	close(release)
	binding, cached, err := cache.getOrAdd(context.Background(), testDatasetIdentity(3, 2, "a"), cachedOfficialTestCase(1, "case-1", "input\n", "ok\n"))
	if err != nil || !cached || binding.fileID != "shared-file" {
		t.Fatalf("shared load after cancellation = %#v/%t/%v", binding, cached, err)
	}
}

func TestSandboxTestcaseCacheEvictsOnlyUnpinnedAstraCodeEntriesByByteBudget(t *testing.T) {
	var deleted []string
	cache := lifecycleTestcaseCache(t, &fakeExecutorRPC{
		fileList: emptyFileList,
		fileAdd:  sequentialFileAdd(),
		fileDelete: func(_ context.Context, fileID *judgepb.FileID) (*emptypb.Empty, error) {
			deleted = append(deleted, fileID.GetFileID())
			return &emptypb.Empty{}, nil
		},
	}, 4, 10, 0)
	identity := testDatasetIdentity(3, 2, "a")
	first := cachedOfficialTestCase(1, "one", "abc", "ok\n")
	second := cachedOfficialTestCase(2, "two", "def", "ok\n")

	firstBinding, cached, err := cache.getOrAdd(context.Background(), identity, first)
	if err != nil || !cached {
		t.Fatalf("first getOrAdd = %#v/%t/%v", firstBinding, cached, err)
	}
	cache.release([]testcaseCacheBinding{firstBinding})
	secondBinding, cached, err := cache.getOrAdd(context.Background(), identity, second)
	if err != nil || !cached {
		t.Fatalf("second getOrAdd = %#v/%t/%v", secondBinding, cached, err)
	}
	cache.release([]testcaseCacheBinding{secondBinding})
	cache.cleanup(context.Background())

	if got, want := fmt.Sprint(deleted), "[file-1]"; got != want {
		t.Fatalf("deleted FileIDs = %s, want %s", got, want)
	}
	firstKey, _ := newTestcaseCacheKey(identity, first)
	secondKey, _ := newTestcaseCacheKey(identity, second)
	if _, ok := cache.lookup(firstKey); ok {
		t.Fatal("least-recently-used unpinned entry remained after max-byte eviction")
	}
	if got, ok := cache.lookup(secondKey); !ok || got != secondBinding.fileID {
		t.Fatalf("newer entry = %q/%t, want retained %q", got, ok, secondBinding.fileID)
	}
}

func TestSandboxTestcaseCacheNeverEvictsPinnedOrForeignFileIDs(t *testing.T) {
	var deleted []string
	cache := lifecycleTestcaseCache(t, &fakeExecutorRPC{
		fileList: func(context.Context, *emptypb.Empty) (*judgepb.FileListType, error) {
			return &judgepb.FileListType{FileIDs: map[string]string{
				"compiled-executable": "main",
				"foreign-input":       "third-party-file",
			}}, nil
		},
		fileAdd: sequentialFileAdd(),
		fileDelete: func(_ context.Context, fileID *judgepb.FileID) (*emptypb.Empty, error) {
			deleted = append(deleted, fileID.GetFileID())
			return &emptypb.Empty{}, nil
		},
	}, 1, 1, 0)
	identity := testDatasetIdentity(3, 2, "a")
	pinnedCase := cachedOfficialTestCase(1, "pinned", "aa", "ok\n")
	otherCase := cachedOfficialTestCase(2, "other", "bb", "ok\n")
	pinned, cached, err := cache.getOrAdd(context.Background(), identity, pinnedCase)
	if err != nil || !cached {
		t.Fatalf("pinned getOrAdd = %#v/%t/%v", pinned, cached, err)
	}
	other, cached, err := cache.getOrAdd(context.Background(), identity, otherCase)
	if err != nil || !cached {
		t.Fatalf("other getOrAdd = %#v/%t/%v", other, cached, err)
	}
	cache.release([]testcaseCacheBinding{other})
	cache.cleanup(context.Background())

	if got, want := fmt.Sprint(deleted), "[file-2]"; got != want {
		t.Fatalf("deleted FileIDs = %s, want only unpinned testcase file %s", got, want)
	}
	if deleted[0] == pinned.fileID || deleted[0] == "compiled-executable" || deleted[0] == "foreign-input" {
		t.Fatalf("unsafe FileDelete target %q", deleted[0])
	}
	cache.release([]testcaseCacheBinding{pinned})
	cache.cleanup(context.Background())
	if got, want := fmt.Sprint(deleted), "[file-2 file-1]"; got != want {
		t.Fatalf("released pinned entry did not later become eligible: %s", got)
	}
}

func TestSandboxTestcaseCacheDeleteFailureKeepsEntryAndDoesNotBreakJudging(t *testing.T) {
	cache := lifecycleTestcaseCache(t, &fakeExecutorRPC{
		fileList: emptyFileList,
		fileAdd:  sequentialFileAdd(),
		fileDelete: func(context.Context, *judgepb.FileID) (*emptypb.Empty, error) {
			return nil, errors.New("sandbox delete unavailable")
		},
	}, 1, 1, 0)
	identity := testDatasetIdentity(3, 2, "a")
	testCase := cachedOfficialTestCase(1, "case", "too-large", "ok\n")
	binding, cached, err := cache.getOrAdd(context.Background(), identity, testCase)
	if err != nil || !cached {
		t.Fatalf("getOrAdd = %#v/%t/%v", binding, cached, err)
	}
	cache.release([]testcaseCacheBinding{binding})
	cache.cleanup(context.Background())
	key, _ := newTestcaseCacheKey(identity, testCase)
	if got, ok := cache.lookup(key); !ok || got != binding.fileID {
		t.Fatalf("entry after failed delete = %q/%t, want retained %q", got, ok, binding.fileID)
	}
}

func TestSandboxTestcaseCacheDisabledUsesMemoryFileWithoutFileRPC(t *testing.T) {
	cache, err := newConfiguredSandboxTestcaseCache(&fakeExecutorRPC{
		fileAdd: func(context.Context, *judgepb.FileContent) (*judgepb.FileID, error) {
			t.Fatal("disabled cache called FileAdd")
			return nil, nil
		},
	}, zap.NewNop(), config.TestcaseCacheConfig{})
	if err != nil {
		t.Fatal(err)
	}
	binding, cached, err := cache.getOrAdd(context.Background(), testDatasetIdentity(3, 2, "a"), cachedOfficialTestCase(1, "case", "input", "ok\n"))
	if err != nil || cached || binding.fileID != "" {
		t.Fatalf("disabled cache getOrAdd = %#v/%t/%v", binding, cached, err)
	}
}

func TestSandboxTestcaseCacheReconciliationUsesV1SizeSuffixWithoutReadingContents(t *testing.T) {
	identity := testDatasetIdentity(3, 2, "a")
	testCase := cachedOfficialTestCase(1, "case", "abcdef", "ok\n")
	legacyCase := cachedOfficialTestCase(2, "legacy", "legacy-input", "ok\n")
	key, ok := newTestcaseCacheKey(identity, testCase)
	if !ok {
		t.Fatal("build cache key")
	}
	legacyKey, ok := newTestcaseCacheKey(identity, legacyCase)
	if !ok {
		t.Fatal("build legacy cache key")
	}
	cache := lifecycleTestcaseCache(t, &fakeExecutorRPC{fileList: func(context.Context, *emptypb.Empty) (*judgepb.FileListType, error) {
		return &judgepb.FileListType{FileIDs: map[string]string{"sized": key.nameWithSize(int64(len(testCase.Stdin))), "legacy": legacyKey.name()}}, nil
	}}, 100, 10, 0)
	if err := cache.reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	cache.mu.RLock()
	entry := cache.entries[key]
	total, unknown := cache.totalBytes, cache.unknownEntries
	cache.mu.RUnlock()
	if entry == nil || !entry.sizeKnown || total != int64(len(testCase.Stdin)) || unknown != 1 {
		t.Fatalf("sized reconciliation entry/accounting = %#v/%d/%d", entry, total, unknown)
	}
}

func TestSandboxTestcaseCacheEvictsIdleEntryBeforeCapacityPressure(t *testing.T) {
	var deleted []string
	cache := lifecycleTestcaseCache(t, &fakeExecutorRPC{
		fileList: emptyFileList,
		fileAdd:  sequentialFileAdd(),
		fileDelete: func(_ context.Context, fileID *judgepb.FileID) (*emptypb.Empty, error) {
			deleted = append(deleted, fileID.GetFileID())
			return &emptypb.Empty{}, nil
		},
	}, 1<<20, 10, time.Minute)
	identity := testDatasetIdentity(3, 2, "a")
	oldCase := cachedOfficialTestCase(1, "old", "old", "ok\n")
	freshCase := cachedOfficialTestCase(2, "fresh", "fresh", "ok\n")
	oldBinding, _, _ := cache.getOrAdd(context.Background(), identity, oldCase)
	cache.release([]testcaseCacheBinding{oldBinding})
	freshBinding, _, _ := cache.getOrAdd(context.Background(), identity, freshCase)
	cache.release([]testcaseCacheBinding{freshBinding})
	oldKey, _ := newTestcaseCacheKey(identity, oldCase)
	cache.mu.Lock()
	cache.entries[oldKey].lastUsed = time.Now().Add(-2 * time.Minute)
	cache.mu.Unlock()
	cache.cleanup(context.Background())
	if got, want := fmt.Sprint(deleted), "[file-1]"; got != want {
		t.Fatalf("idle eviction FileIDs = %s, want %s", got, want)
	}
	if _, ok := cache.lookup(oldKey); ok {
		t.Fatal("idle entry remained after cleanup")
	}
}

func TestSandboxTestcaseCacheStaleInvalidationClearsAccounting(t *testing.T) {
	cache := lifecycleTestcaseCache(t, &fakeExecutorRPC{fileList: emptyFileList, fileAdd: sequentialFileAdd()}, 1024, 10, 0)
	identity := testDatasetIdentity(3, 2, "a")
	testCase := cachedOfficialTestCase(1, "case", "input", "ok\n")
	binding, cached, err := cache.getOrAdd(context.Background(), identity, testCase)
	if err != nil || !cached {
		t.Fatalf("getOrAdd = %#v/%t/%v", binding, cached, err)
	}
	cache.release([]testcaseCacheBinding{binding})
	if !cache.invalidateMissing(context.Background(), []testcaseCacheBinding{binding}) {
		t.Fatal("missing sandbox FileID was not invalidated")
	}
	cache.mu.RLock()
	entries, bytes := len(cache.entries), cache.totalBytes
	cache.mu.RUnlock()
	if entries != 0 || bytes != 0 {
		t.Fatalf("stale invalidation accounting = entries=%d bytes=%d, want zero", entries, bytes)
	}
}

func TestSandboxTestcaseCacheConcurrentPinnedUseAndCleanup(t *testing.T) {
	var deletes atomic.Int32
	cache := lifecycleTestcaseCache(t, &fakeExecutorRPC{
		fileList: emptyFileList,
		fileAdd:  sequentialFileAdd(),
		fileDelete: func(context.Context, *judgepb.FileID) (*emptypb.Empty, error) {
			deletes.Add(1)
			return &emptypb.Empty{}, nil
		},
	}, 1, 1, 0)
	identity := testDatasetIdentity(3, 2, "a")
	binding, cached, err := cache.getOrAdd(context.Background(), identity, cachedOfficialTestCase(1, "case", "input", "ok\n"))
	if err != nil || !cached {
		t.Fatalf("getOrAdd = %#v/%t/%v", binding, cached, err)
	}
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.cleanup(context.Background())
		}()
	}
	wg.Wait()
	if got := deletes.Load(); got != 0 {
		t.Fatalf("cleanup deleted pinned testcase file %d times", got)
	}
	cache.release([]testcaseCacheBinding{binding})
	cache.cleanup(context.Background())
	if got := deletes.Load(); got != 1 {
		t.Fatalf("released testcase file deletes = %d, want one", got)
	}
}

func TestGoJudgeClientPinsCachedInputUntilBatchExecReturns(t *testing.T) {
	runStarted := make(chan struct{})
	allowRunReturn := make(chan struct{})
	rpc := &fakeExecutorRPC{
		fileList: emptyFileList,
		fileAdd:  sequentialFileAdd(),
		handler: func(_ context.Context, request *judgepb.Request) (*judgepb.Response, error) {
			if isCompileRequest(request) {
				return compileAccepted("exe"), nil
			}
			close(runStarted)
			<-allowRunReturn
			return acceptedResults("ok\n"), nil
		},
	}
	client := NewGoJudgeClient(rpc, zap.NewNop(), config.TestcaseCacheConfig{
		Enabled: true, MaxBytes: 1, MaxEntries: 1, CleanupInterval: time.Hour,
	})
	client.testcaseCache.nextCleanup = time.Now().Add(time.Hour)
	done := make(chan error, 1)
	go func() {
		_, err := client.Execute(context.Background(), outbound.ExecutionRequest{
			Language: "CPP", SourceCode: "int main(){}", StopOnFirstFailure: true,
			TestcaseDataset: testDatasetIdentity(3, 2, "a"),
			TestCases:       []outbound.ExecutionTestCase{cachedOfficialTestCase(1, "case", "input", "ok\n")},
		})
		done <- err
	}()
	<-runStarted
	client.testcaseCache.cleanup(context.Background())
	if got := rpc.deletedFileIDs(); len(got) != 0 {
		t.Fatalf("cleanup deleted a file while Exec was active: %v", got)
	}
	close(allowRunReturn)
	if err := <-done; err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := rpc.deletedFileIDs(); fmt.Sprint(got) != "[exe]" {
		t.Fatalf("post-execution cleanup = %v, want only the submission executable", got)
	}
	client.testcaseCache.cleanup(context.Background())
	if got := rpc.deletedFileIDs(); len(got) != 2 || got[0] != "exe" || got[1] == "exe" {
		t.Fatalf("cleanup after Exec returned deleted %v, want executable then testcase cache file", got)
	}
}

func lifecycleTestcaseCache(t *testing.T, rpc sandboxFileRPC, maxBytes int64, maxEntries int, idleTTL time.Duration) *sandboxTestcaseCache {
	t.Helper()
	cache, err := newConfiguredSandboxTestcaseCache(rpc, zap.NewNop(), config.TestcaseCacheConfig{
		Enabled: true, MaxBytes: maxBytes, MaxEntries: maxEntries, IdleTTL: idleTTL, CleanupInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("create lifecycle cache: %v", err)
	}
	// Tests invoke cleanup explicitly. Prevent a release from scheduling a
	// concurrent cleanup that would make the assertions timing-dependent.
	cache.nextCleanup = time.Now().Add(time.Hour)
	return cache
}

func assertCachedOfficialStdin(t *testing.T, request *judgepb.Request, fileID string) {
	t.Helper()
	for _, command := range request.GetCmd() {
		if command.GetFiles()[0].GetMemory() != nil {
			t.Fatal("cached testcase stdin unexpectedly included MemoryFile content")
		}
		if got := command.GetFiles()[0].GetCached().GetFileID(); got != fileID {
			t.Fatalf("cached testcase FileID = %q, want %q", got, fileID)
		}
	}
}

func testDatasetIdentity(problemID int64, version int, fill string) *outbound.TestcaseDatasetIdentity {
	return &outbound.TestcaseDatasetIdentity{ProblemID: problemID, Version: version, DatasetChecksum: strings.Repeat(fill, 64)}
}

func cachedOfficialTestCase(index int, id, stdin, expected string) outbound.ExecutionTestCase {
	return outbound.ExecutionTestCase{Index: index, ID: id, Kind: "official", Stdin: stdin, ExpectedOutput: &expected}
}

func emptyFileList(context.Context, *emptypb.Empty) (*judgepb.FileListType, error) {
	return &judgepb.FileListType{}, nil
}

func sequentialFileAdd() func(context.Context, *judgepb.FileContent) (*judgepb.FileID, error) {
	var calls atomic.Int32
	return func(context.Context, *judgepb.FileContent) (*judgepb.FileID, error) {
		return &judgepb.FileID{FileID: fmt.Sprintf("file-%d", calls.Add(1))}, nil
	}
}

func enabledTestcaseCacheConfig() config.TestcaseCacheConfig {
	return config.TestcaseCacheConfig{
		Enabled: true, MaxBytes: 1 << 30, MaxEntries: 1_000,
		CleanupInterval: time.Hour,
	}
}

func newSandboxTestcaseCache(client sandboxFileRPC, logger *zap.Logger) *sandboxTestcaseCache {
	cache, err := newConfiguredSandboxTestcaseCache(client, logger, config.TestcaseCacheConfig{
		Enabled: true, MaxBytes: math.MaxInt64, MaxEntries: math.MaxInt, CleanupInterval: time.Hour,
	})
	if err != nil {
		panic(err)
	}
	return cache
}
