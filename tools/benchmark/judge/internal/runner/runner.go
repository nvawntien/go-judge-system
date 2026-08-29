// Package runner coordinates preflight, optional pre-warmup refresh, warmup,
// open-loop load, SSE-first completion, drain, and result persistence.
package runner

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"math/rand/v2"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/client"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/config"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/credentials"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/model"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/observer"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/report"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/resources"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/result"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/scheduler"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/stats"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/systemconfig"
)

type Prepared struct {
	Config               config.Config
	Sessions             []*credentials.Session
	Clients              map[string]*client.API
	Source               []byte
	SourceText           string
	SourceSHA256         string
	Subjects             map[string]string // alias -> real ID; memory-only
	RequiredUsers        int
	ConfiguredUsers      int
	NoFile               resources.NoFileStatus
	TransportDiagnostics *client.Diagnostics
	validationIndexes    []int
	closeTransport       func()
}

// maxSourceBytes mirrors the public submission contract (256 KiB) without
// importing a production service internal package.
const maxSourceBytes = 256 * 1024

// Preflight performs filesystem checks, authenticated GETs, and at most one
// normal Auth refresh per session when the configured refresh policy requires
// it before identity validation. It never creates a submission or ticket.
func Preflight(ctx context.Context, cfg config.Config) (*Prepared, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if cfg.SystemConfigFile != "" {
		value, err := systemconfig.Load(cfg.SystemConfigFile)
		if err != nil {
			return nil, err
		}
		cfg.SystemConfig = value
	}
	file, err := credentials.Load(cfg.UsersFile)
	if err != nil {
		return nil, err
	}
	configuredUsers := len(file.Users)
	if cfg.UserCount > 0 {
		file, err = credentials.SelectCanonical(file, cfg.UserCount)
		if err != nil {
			return nil, err
		}
	}
	noFile, err := resources.CheckNoFileForObservation(cfg.MaxInFlight, noFileObservation(cfg.ObservationMode))
	if err != nil {
		return nil, err
	}
	transport := credentials.NewBenchmarkTransport(int(noFile.Required))
	sessions, err := credentials.NewSessions(file, cfg.BaseURL, transport)
	if err != nil {
		return nil, err
	}
	sourceInfo, err := os.Lstat(cfg.SourceFile)
	if err != nil || sourceInfo.Mode()&os.ModeSymlink != 0 || !sourceInfo.Mode().IsRegular() {
		return nil, errors.New("source file must be a regular readable file")
	}
	if sourceInfo.Size() <= 0 || sourceInfo.Size() > maxSourceBytes {
		return nil, fmt.Errorf("source file must be between 1 and %d bytes", maxSourceBytes)
	}
	source, err := os.ReadFile(cfg.SourceFile)
	if err != nil {
		return nil, fmt.Errorf("read source file: %w", err)
	}
	hash := sha256.Sum256(source)
	diagnostics := &client.Diagnostics{}
	prepared := &Prepared{Config: cfg, Sessions: sessions, Clients: make(map[string]*client.API, len(sessions)), Source: source, SourceText: string(source), SourceSHA256: fmt.Sprintf("%x", hash[:]), Subjects: make(map[string]string), ConfiguredUsers: configuredUsers, NoFile: noFile, TransportDiagnostics: diagnostics, closeTransport: transport.CloseIdleConnections}
	keepTransport := false
	defer func() {
		if !keepTransport && prepared.closeTransport != nil {
			prepared.closeTransport()
		}
	}()
	for _, session := range sessions {
		api, err := client.NewWithDiagnostics(cfg.BaseURL, session, diagnostics)
		if err != nil {
			return nil, err
		}
		prepared.Clients[session.Alias] = api
	}
	prepared.validationIndexes = sessionValidationIndexes(cfg, len(sessions))
	subjects, err := validatePreflightSessions(ctx, prepared, prepared.validationIndexes)
	if err != nil {
		return nil, err
	}
	for index, subject := range subjects {
		session := sessions[index]
		session.Subject = subject
		prepared.Subjects[session.Alias] = subject
	}
	problem, err := publicProblemWithTimeout(ctx, prepared.Clients[sessions[0].Alias], cfg.APITimeout, cfg.ProblemSlug)
	if err != nil {
		prepared.closeTransport()
		return nil, fmt.Errorf("preflight public problem: %w", err)
	}
	if problem.ID != cfg.ProblemID {
		prepared.closeTransport()
		return nil, fmt.Errorf("problem slug resolved to ID %d, want %d", problem.ID, cfg.ProblemID)
	}
	if cfg.Mode == config.ModeSustained {
		required, err := scheduler.RequiredUsers(cfg.Rate, cfg.SubmitCooldown, cfg.CooldownGuard, cfg.SubmitLatencyBudget, cfg.PoolHeadroomPercent)
		if err != nil {
			prepared.closeTransport()
			return nil, err
		}
		prepared.RequiredUsers = required
		if len(sessions) < required {
			prepared.closeTransport()
			return nil, fmt.Errorf("benchmark user pool has %d users, need at least %d", len(sessions), required)
		}
	} else if len(sessions) < cfg.BurstSize {
		return nil, fmt.Errorf("benchmark user pool has %d users, need %d distinct burst users", len(sessions), cfg.BurstSize)
	}
	if err := checkResultRoot(cfg.ResultRoot); err != nil {
		return nil, err
	}
	keepTransport = true
	return prepared, nil
}

// PrepareSessions makes one final bounded refresh/lifetime decision and
// revalidates the preflight-authenticated identity immediately before warmup.
// It remains before measured load/drain and never writes refreshed cookies to
// users.local.json. This second pass matters for a large pool whose initial
// preflight itself consumed meaningful access-token lifetime.
func PrepareSessions(ctx context.Context, prepared *Prepared) error {
	cfg := prepared.Config
	return parallelSessionIndexes(ctx, prepared.Sessions, prepared.validationIndexes, cfg.PreflightConcurrency, func(index int, session *credentials.Session) error {
		api := prepared.Clients[session.Alias]
		if err := prepareAccessSessionFinalFor(ctx, cfg, len(prepared.Sessions), session, api); err != nil {
			return fmt.Errorf("prepared session %q: %w", session.Alias, err)
		}
		me, err := meWithTimeout(ctx, api, cfg.APITimeout)
		if err != nil || me.ID != prepared.Subjects[session.Alias] || !me.IsActive || me.Role != "user" {
			return fmt.Errorf("prepared session %q no longer has the preflight identity", session.Alias)
		}
		return nil
	})
}

func validatePreflightSessions(ctx context.Context, prepared *Prepared, indexes []int) (map[int]string, error) {
	subjects := make(map[int]string, len(indexes))
	var subjectsMu sync.Mutex
	err := parallelSessionIndexes(ctx, prepared.Sessions, indexes, prepared.Config.PreflightConcurrency, func(index int, session *credentials.Session) error {
		api := prepared.Clients[session.Alias]
		if err := prepareAccessSessionFor(ctx, prepared.Config, len(prepared.Sessions), session, api); err != nil {
			return fmt.Errorf("preflight session %q: %w", session.Alias, err)
		}
		me, err := meWithTimeout(ctx, api, prepared.Config.APITimeout)
		if err != nil {
			return fmt.Errorf("preflight session %q: %w", session.Alias, err)
		}
		if me.ID == "" || !me.IsActive || me.Role != "user" {
			return fmt.Errorf("benchmark session %q is not an active normal user", session.Alias)
		}
		subjectsMu.Lock()
		subjects[index] = me.ID
		subjectsMu.Unlock()
		return nil
	})
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(subjects))
	for _, subject := range subjects {
		if _, duplicate := seen[subject]; duplicate {
			return nil, errors.New("benchmark sessions resolve to duplicate authenticated identities")
		}
		seen[subject] = struct{}{}
	}
	return subjects, nil
}

// sessionValidationIndexes keeps existing full-pool validation unchanged.  An
// Admission-only and realistic 100k bursts verify a deterministic sample plus
// first/last canonical identities; bootstrap already validated every session
// while creating the credential file. This avoids an unmeasured second 100k
// authenticated GET storm immediately before the released burst.
func sessionValidationIndexes(cfg config.Config, total int) []int {
	if total == 0 {
		return nil
	}
	if cfg.ObservationMode != config.ObservationAdmissionOnly && cfg.ObservationMode != config.ObservationRealistic {
		indexes := make([]int, total)
		for i := range indexes {
			indexes[i] = i
		}
		return indexes
	}
	want := cfg.AdmissionPreflightSample
	if want < 2 {
		want = 2
	}
	if want > total {
		want = total
	}
	chosen := map[int]struct{}{0: {}, total - 1: {}}
	rng := rand.New(rand.NewPCG(uint64(cfg.Seed), uint64(cfg.Seed)^0x9e3779b97f4a7c15))
	for len(chosen) < want {
		chosen[rng.IntN(total)] = struct{}{}
	}
	indexes := make([]int, 0, len(chosen))
	for index := range chosen {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	return indexes
}

func parallelSessions(ctx context.Context, sessions []*credentials.Session, concurrency int, work func(int, *credentials.Session) error) error {
	indexes := make([]int, len(sessions))
	for i := range indexes {
		indexes[i] = i
	}
	return parallelSessionIndexes(ctx, sessions, indexes, concurrency, work)
}

func parallelSessionIndexes(ctx context.Context, sessions []*credentials.Session, indexes []int, concurrency int, work func(int, *credentials.Session) error) error {
	if concurrency < 1 {
		return errors.New("preflight concurrency must be positive")
	}
	if len(indexes) == 0 {
		return nil
	}
	if concurrency > len(indexes) {
		concurrency = len(indexes)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan int)
	errCh := make(chan error, 1)
	var workers sync.WaitGroup
	for range concurrency {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case index, ok := <-jobs:
					if !ok {
						return
					}
					if err := work(index, sessions[index]); err != nil {
						select {
						case errCh <- err:
							cancel()
						default:
						}
						return
					}
				}
			}
		}()
	}
	for _, index := range indexes {
		select {
		case <-ctx.Done():
			close(jobs)
			workers.Wait()
			select {
			case err := <-errCh:
				return err
			default:
				return ctx.Err()
			}
		case jobs <- index:
		}
	}
	close(jobs)
	workers.Wait()
	select {
	case err := <-errCh:
		return err
	default:
		return ctx.Err()
	}
}

type RunResult struct {
	Dir     string
	Summary model.RunSummary
}

func Run(ctx context.Context, prepared *Prepared) (*RunResult, error) {
	if prepared.closeTransport != nil {
		defer prepared.closeTransport()
	}
	if err := PrepareSessions(ctx, prepared); err != nil {
		return nil, err
	}
	cfg := prepared.Config
	if cfg.RunID == "" {
		cfg.RunID = generatedRunID(cfg)
	}
	// All records must carry the generated stable ID; retain it in the prepared
	// immutable run configuration before any goroutine is launched.
	prepared.Config = cfg
	writer, err := result.Create(cfg.ResultRoot, cfg.RunID)
	if err != nil {
		return nil, err
	}
	run := newRun(ctx, prepared, writer)
	run.metadata = runMetadata(cfg, prepared.SourceSHA256, int64(len(prepared.Source)))
	run.metadata.Users = model.UserSet{Configured: prepared.ConfiguredUsers, Selected: len(prepared.Sessions), OneSubmitPerUser: cfg.Objective == config.ObjectiveMassiveBurst}
	run.metadata.BenchmarkObjective = string(cfg.Objective)
	run.metadata.ObservationMode = string(cfg.ObservationMode)
	run.metadata.SessionValidation = model.SessionValidation{Mode: string(cfg.ObservationMode), Validated: len(prepared.validationIndexes), SampleRequested: cfg.AdmissionPreflightSample, FirstAndLastIncluded: len(prepared.validationIndexes) > 0}
	run.metadata.ClientDiagnostics.NoFileSoftLimit = prepared.NoFile.SoftLimit
	run.metadata.ClientDiagnostics.NoFileHardLimit = prepared.NoFile.HardLimit
	run.metadata.ClientDiagnostics.NoFileRequired = prepared.NoFile.Required
	run.metadata.ClientDiagnostics.NoFileRecommended = prepared.NoFile.Recommended
	run.metadata.ClientDiagnostics.ConnectionsPerLogicalSubmission = prepared.NoFile.ConnectionsPerLogicalSubmission
	run.metadata.ClientDiagnostics.RuntimeCPUCount = runtime.NumCPU()
	if cfg.Mode == config.ModeBurst {
		burstSize, jitter := cfg.BurstSize, cfg.Jitter.Milliseconds()
		run.metadata.Workload.BurstSize = &burstSize
		run.metadata.Workload.JitterMilliseconds = &jitter
	} else {
		rate := rateFloat(cfg.Rate)
		run.metadata.Workload.TargetRatePerSecond = &rate
		if cfg.TotalSubmissions > 0 {
			total := cfg.TotalSubmissions
			run.metadata.Workload.TotalSubmissions = &total
		} else {
			duration := cfg.Duration.Milliseconds()
			run.metadata.Workload.ArrivalDurationMS = &duration
		}
	}
	if err := writer.WriteRun(run.metadata); err != nil {
		return nil, err
	}
	run.startClientSampler()
	if err := run.warmup(); err != nil {
		run.abort("WARMUP_FAILED")
		finished, finishErr := run.finish()
		if finishErr != nil {
			return finished, fmt.Errorf("warmup failed: %w; persist results: %v", err, finishErr)
		}
		return finished, err
	}
	if cfg.Mode == config.ModeBurst {
		run.burst()
	} else {
		run.sustained()
	}
	return run.finish()
}

type benchmarkRun struct {
	parentCtx                  context.Context
	ctx                        context.Context
	cancel                     context.CancelFunc
	prepared                   *Prepared
	writer                     *result.Writer
	pool                       *scheduler.Pool
	limiter                    *observer.ReconcileLimiter
	redactor                   result.Redactor
	metadata                   model.RunMetadata
	mu                         sync.Mutex
	records                    []*model.SubmissionRecord
	acceptedIDs                map[int64]struct{}
	quality                    map[string]struct{}
	postWG                     sync.WaitGroup
	observerWG                 sync.WaitGroup
	activePosts                int
	activeTickets              int
	activeSSE                  int
	outstanding                int
	peakInFlight               int
	peakLogicalInFlight        int
	activeObservers            int
	peakObservers              int
	peakPosts                  int
	peakTickets                int
	peakSSE                    int
	clientSamples              []model.ClientResourceSample
	clientSamplerStop          chan struct{}
	clientSamplerDone          chan struct{}
	burstGoroutinesBefore      int
	burstGoroutinesAfterLaunch int
	burstLaunchCompletion      *int64
	stopArrivals               bool
	loadStart                  time.Time
	loadEnd                    time.Time
	drainStart                 time.Time
	drainEnd                   time.Time
}

func newRun(ctx context.Context, prepared *Prepared, writer *result.Writer) *benchmarkRun {
	aliases := make([]string, 0, len(prepared.Sessions))
	for _, session := range prepared.Sessions {
		aliases = append(aliases, session.Alias)
	}
	pool, _ := scheduler.NewPool(aliases, uint64(prepared.Config.Seed))
	runContext, cancel := context.WithCancel(ctx)
	secrets := []string{prepared.SourceText}
	// Admission-only never stores tickets or URLs carrying credentials, so do
	// not retain 100k cookie copies in the load generator merely for redaction.
	if prepared.Config.ObservationMode == config.ObservationFull {
		for _, session := range prepared.Sessions {
			for _, cookie := range session.Jar.Cookies(prepared.Config.BaseURL) {
				secrets = append(secrets, cookie.Value)
			}
		}
	}
	for _, subject := range prepared.Subjects {
		secrets = append(secrets, subject)
	}
	return &benchmarkRun{parentCtx: ctx, ctx: runContext, cancel: cancel, prepared: prepared, writer: writer, pool: pool, limiter: observer.NewReconcileLimiter(prepared.Config.ReconcileMaxQPS), redactor: result.NewRedactor(secrets...), quality: map[string]struct{}{}, acceptedIDs: map[int64]struct{}{}}
}

func noFileObservation(mode config.ObservationMode) resources.Observation {
	switch mode {
	case config.ObservationAdmissionOnly:
		return resources.ObservationAdmissionOnly
	case config.ObservationRealistic:
		return resources.ObservationRealistic
	default:
		return resources.ObservationTerminal
	}
}

// Client resource sampling is intentionally low frequency. It records local
// evidence only, and it never changes host limits or reads target state.
func (r *benchmarkRun) startClientSampler() {
	if r.prepared.Config.Mode != config.ModeBurst {
		return
	}
	r.clientSamplerStop, r.clientSamplerDone = make(chan struct{}), make(chan struct{})
	go func() {
		defer close(r.clientSamplerDone)
		r.captureClientSample()
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-r.clientSamplerStop:
				r.captureClientSample()
				return
			case <-ticker.C:
				r.captureClientSample()
			}
		}
	}()
}

func (r *benchmarkRun) stopClientSampler() {
	if r.clientSamplerStop == nil {
		return
	}
	close(r.clientSamplerStop)
	<-r.clientSamplerDone
	r.clientSamplerStop, r.clientSamplerDone = nil, nil
}

func (r *benchmarkRun) captureClientSample() {
	openFDs, status := localOpenFDs()
	r.mu.Lock()
	sample := model.ClientResourceSample{At: time.Now().UTC(), OpenFDs: openFDs, Goroutines: runtime.NumGoroutine(), ActivePosts: r.activePosts, ActiveTickets: r.activeTickets, ActiveSSEStreams: r.activeSSE}
	r.clientSamples = append(r.clientSamples, sample)
	if openFDs > r.metadata.ClientDiagnostics.PeakOpenFDs {
		r.metadata.ClientDiagnostics.PeakOpenFDs = openFDs
	}
	r.metadata.ClientDiagnostics.OpenFDSamples++
	if status != "" {
		r.metadata.ClientDiagnostics.OpenFDStatus = status
	}
	r.mu.Unlock()
}

func localOpenFDs() (int, string) {
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		return 0, "UNAVAILABLE"
	}
	return len(entries), "AVAILABLE"
}

func (r *benchmarkRun) warmup() error {
	warmupStart := time.Now().UTC()
	r.metadata.Phases.Warmup.StartedAt = warmupStart
	defer func() {
		ended := time.Now().UTC()
		r.metadata.Phases.Warmup.EndedAt = &ended
	}()
	if r.prepared.Config.WarmupCount == 0 {
		return nil
	}
	deadline, cancel := context.WithTimeout(r.ctx, r.prepared.Config.WarmupTimeout)
	defer cancel()
	for index := 0; index < r.prepared.Config.WarmupCount; index++ {
		now := time.Now()
		record, done := r.launch(deadline, model.PhaseWarmup, index, now, "")
		select {
		case <-deadline.Done():
			return errors.New("warmup deadline exceeded")
		case <-done:
		}
		if record.Outcome != model.OutcomeTerminal || record.TerminalStatus != r.prepared.Config.ExpectedVerdict {
			return fmt.Errorf("warmup submission %d did not reach expected verdict", index+1)
		}
	}
	if r.prepared.Config.Mode == config.ModeSustained {
		observed := warmupP95(r.records)
		budget := r.prepared.Config.SubmitLatencyBudget
		if observed > budget {
			budget = observed
		}
		required, err := scheduler.RequiredUsers(r.prepared.Config.Rate, r.prepared.Config.SubmitCooldown, r.prepared.Config.CooldownGuard, budget, r.prepared.Config.PoolHeadroomPercent)
		if err != nil || len(r.prepared.Sessions) < required {
			return errors.New("user pool is insufficient after warmup")
		}
	}
	for r.countEligible(time.Now()) < r.requiredEligible() {
		if !sleepContext(deadline, 10*time.Millisecond) {
			return errors.New("warmup cooldown did not clear before deadline")
		}
	}
	return nil
}

func (r *benchmarkRun) requiredEligible() int {
	if r.prepared.Config.Mode == config.ModeBurst {
		return r.prepared.Config.BurstSize
	}
	return r.prepared.RequiredUsers
}

func (r *benchmarkRun) burst() {
	cfg := r.prepared.Config
	aliases := make([]string, 0, len(r.prepared.Sessions))
	for _, s := range r.prepared.Sessions {
		aliases = append(aliases, s.Alias)
	}
	plan, err := scheduler.BurstPlan(aliases, cfg.BurstSize, uint64(cfg.Seed), cfg.Jitter)
	if err != nil {
		r.addQuality("BURST_PLAN_FAILED")
		return
	}
	// The complete deterministic plan is prepared before this release point.
	// A single coordinator avoids one arrival goroutine per configured user;
	// launch itself returns immediately after it reserves a bounded POST slot.
	sort.SliceStable(plan, func(i, j int) bool { return plan[i].Offset < plan[j].Offset })
	r.loadStart = time.Now()
	r.burstGoroutinesBefore = runtime.NumGoroutine()
	r.metadata.Phases.Load.StartedAt = r.loadStart.UTC()
	for index, arrival := range plan {
		intended := r.loadStart.Add(arrival.Offset)
		if !sleepUntil(r.ctx, intended) {
			r.recordCancelled(model.PhaseLoad, index, arrival.Alias, intended)
			continue
		}
		if r.arrivalMissed(intended) && cfg.Objective != config.ObjectiveMassiveBurst {
			r.recordMissed(model.PhaseLoad, index, arrival.Alias, intended)
			continue
		}
		if r.arrivalMissed(intended) {
			r.addQuality("LOAD_GENERATOR_LIMITED")
		}
		r.launch(r.ctx, model.PhaseLoad, index, intended, arrival.Alias)
	}
	r.loadEnd = time.Now()
	r.burstGoroutinesAfterLaunch = runtime.NumGoroutine()
	launchCompletion := r.loadEnd.Sub(r.loadStart).Milliseconds()
	r.burstLaunchCompletion = &launchCompletion
	loadEnd := r.loadEnd.UTC()
	r.metadata.Phases.Load.EndedAt = &loadEnd
}

func (r *benchmarkRun) sustained() {
	cfg := r.prepared.Config
	r.loadStart = time.Now()
	r.metadata.Phases.Load.StartedAt = r.loadStart.UTC()
	var count int
	var err error
	if cfg.TotalSubmissions > 0 {
		count, err = scheduler.ExactArrivalCount(cfg.TotalSubmissions)
	} else {
		count, err = scheduler.ArrivalCount(cfg.Rate, cfg.Duration)
	}
	if err != nil {
		r.addQuality("SCHEDULER_PLAN_FAILED")
		count = 0
	}
	for index := 0; index < count; index++ {
		offset := scheduler.OffsetFor(cfg.Rate, index)
		intended := r.loadStart.Add(offset)
		if !sleepUntil(r.ctx, intended) {
			r.recordCancelled(model.PhaseLoad, index, "", intended)
			break
		}
		if r.arrivalMissed(intended) {
			r.recordMissed(model.PhaseLoad, index, "", intended)
			continue
		}
		r.launch(r.ctx, model.PhaseLoad, index, intended, "")
	}
	if cfg.TotalSubmissions > 0 {
		// Exact-volume mode ends as soon as logical arrival N has been created;
		// there is intentionally no synthetic final sleep based on N/rate.
		r.loadEnd = time.Now()
	} else {
		// Duration mode retains its existing half-open configured boundary.
		boundary := r.loadStart.Add(cfg.Duration)
		sleepUntil(r.ctx, boundary)
		r.loadEnd = boundary
	}
	loadEnd := r.loadEnd.UTC()
	r.metadata.Phases.Load.EndedAt = &loadEnd
}

func (r *benchmarkRun) launch(ctx context.Context, phase model.Phase, sequence int, intended time.Time, preferredAlias string) (*model.SubmissionRecord, <-chan struct{}) {
	done := make(chan struct{})
	record := &model.SubmissionRecord{RunID: r.prepared.Config.RunID, Phase: phase, Sequence: sequence, Outcome: model.OutcomeLocalFailure}
	record.IntendedAt = &intended
	if phase == model.PhaseLoad {
		offset := intended.Sub(r.loadStart).Milliseconds()
		record.IntendedOffsetMS = &offset
	}
	r.mu.Lock()
	if r.stopArrivals && phase == model.PhaseLoad {
		r.mu.Unlock()
		r.recordCancelledPointer(record)
		close(done)
		return record, done
	}
	now := time.Now()
	if r.outstanding+r.activePosts >= r.prepared.Config.MaxInFlight {
		r.addQualityLocked("MAX_IN_FLIGHT_REACHED")
		r.records = append(r.records, record)
		r.mu.Unlock()
		r.recordFailure(record, model.OutcomeLocalFailure, "max_in_flight", "max in-flight limit reached")
		close(done)
		return record, done
	}
	var user *scheduler.User
	var ok bool
	if preferredAlias != "" {
		user, ok = r.pool.LeaseAlias(preferredAlias, now)
	} else {
		user, ok = r.pool.Lease(now)
	}
	if !ok {
		r.addQualityLocked("USER_POOL_EXHAUSTED")
		r.records = append(r.records, record)
		r.mu.Unlock()
		r.recordFailure(record, model.OutcomeUserExhausted, "scheduler_user_exhausted", "no benchmark user was eligible at intended arrival")
		close(done)
		return record, done
	}
	record.UserAlias = user.Alias
	r.records = append(r.records, record)
	r.activePosts++
	if r.activePosts > r.peakPosts {
		r.peakPosts = r.activePosts
	}
	r.updatePeakLogicalInFlightLocked()
	r.postWG.Add(1)
	r.mu.Unlock()
	go func() {
		defer r.postWG.Done()
		started := time.Now()
		r.setStarted(record, started)
		requestCtx, cancel := context.WithTimeout(ctx, r.prepared.Config.APITimeout)
		response, err := r.prepared.Clients[user.Alias].Submit(requestCtx, client.SubmissionRequest{ProblemID: r.prepared.Config.ProblemID, Language: r.prepared.Config.Language, SourceCode: r.prepared.SourceText})
		cancel()
		completed := time.Now()
		r.setCompleted(record, completed)
		r.mu.Lock()
		r.activePosts--
		r.mu.Unlock()
		if err != nil {
			if response.HTTPStatus != 0 {
				r.mu.Lock()
				record.HTTPStatus = intPtr(response.HTTPStatus)
				record.APICode = intPtr(response.APICode)
				r.mu.Unlock()
			}
			r.poolQuarantine(user.Alias, completed)
			errorClass := client.TransportErrorClass(err)
			r.recordFailure(record, model.OutcomeAmbiguousPost, errorClass, err.Error())
			if errorClass == "client_emfile" || errorClass == "client_enfile" || errorClass == "client_ephemeral_port_exhaustion" {
				r.addQuality("LOAD_GENERATOR_LIMITED")
			}
			r.addQuality("AMBIGUOUS_POST")
			close(done)
			return
		}
		switch response.Kind {
		case client.SubmitAccepted:
			id := response.Submission.ID
			r.mu.Lock()
			if _, duplicate := r.acceptedIDs[id]; duplicate {
				record.HTTPStatus = intPtr(response.HTTPStatus)
				record.APICode = intPtr(response.APICode)
				record.SubmissionID = &id
				record.Outcome = model.OutcomeLocalFailure
				record.ErrorClass = "duplicate_submission_id"
				record.Error = "server returned a duplicate submission ID"
				r.addQualityLocked("DATA_INTEGRITY_FAILURE")
				r.mu.Unlock()
				r.poolAccepted(user.Alias, completed)
				close(done)
				return
			}
			r.acceptedIDs[id] = struct{}{}
			record.Accepted = true
			record.HTTPStatus = intPtr(response.HTTPStatus)
			record.HTTPProtocol = response.Protocol
			record.APICode = intPtr(response.APICode)
			record.SubmissionID = &id
			record.InitialStatus = response.Submission.Status
			if r.prepared.Config.ObservationMode == config.ObservationFull {
				r.outstanding++
				if r.outstanding > r.peakInFlight {
					r.peakInFlight = r.outstanding
				}
				r.updatePeakLogicalInFlightLocked()
			} else {
				record.Outcome = model.OutcomeAccepted
			}
			r.mu.Unlock()
			r.poolAccepted(user.Alias, completed)
			if r.prepared.Config.ObservationMode == config.ObservationAdmissionOnly {
				close(done)
				return
			}
			r.observerWG.Add(1)
			r.mu.Lock()
			r.activeObservers++
			if r.activeObservers > r.peakObservers {
				r.peakObservers = r.activeObservers
			}
			r.mu.Unlock()
			go func() {
				defer r.observerWG.Done()
				defer func() {
					r.mu.Lock()
					r.activeObservers--
					r.mu.Unlock()
				}()
				defer close(done)
				observerStarted := time.Now()
				r.mu.Lock()
				record.ObserverStartedAt = &observerStarted
				r.mu.Unlock()
				if r.prepared.Config.ObservationMode == config.ObservationRealistic {
					r.observeRealistic(ctx, record, id)
					observerEnded := time.Now()
					r.mu.Lock()
					record.ObserverEndedAt = &observerEnded
					r.mu.Unlock()
					return
				}
				obs := observer.Observe(ctx, r.prepared.Clients[user.Alias], id, observer.Config{ConnectTimeout: r.prepared.Config.SSEConnectTimeout, IdleTimeout: r.prepared.Config.SSEIdleTimeout, SubmissionTimeout: r.prepared.Config.SubmissionTimeout, MaxReconnects: r.prepared.Config.SSEMaxReconnects, BackoffBase: r.prepared.Config.SSEBackoffBase, BackoffMax: r.prepared.Config.SSEBackoffMax, SafetyReconcileInterval: r.prepared.Config.SafetyReconcileInterval}, r.limiter)
				observerEnded := time.Now()
				r.mu.Lock()
				record.ObserverEndedAt = &observerEnded
				record.SSEConnections, record.SSEFailures, record.GETReconciliations = obs.SSEConnections, obs.SSEFailures, obs.GETReconciliations
				if obs.TerminalStatus != "" {
					status := obs.TerminalStatus
					at := obs.ObservedAt
					record.TerminalStatus = status
					record.TerminalObservedAt = &at
					record.CompletionSource = obs.Source
					offset := at.Sub(r.loadStart).Milliseconds()
					record.TerminalOffsetMS = &offset
					e2e := at.Sub(started).Seconds() * 1000
					acceptedToTerminal := at.Sub(completed).Seconds() * 1000
					record.EndToEndLatencyMS = &e2e
					record.AcceptedToTerminalMS = &acceptedToTerminal
					record.Outcome = model.OutcomeTerminal
					r.outstanding--
					if status != r.prepared.Config.ExpectedVerdict {
						r.addQualityLocked("WORKLOAD_MISMATCH")
					}
				} else if obs.AuthFailure {
					record.Outcome = model.OutcomeAuthFailure
					r.poolDisableLocked(user.Alias)
					r.addQualityLocked("AUTH_FAILURE")
				} else {
					record.Outcome = model.OutcomeCompletionTimeout
					r.addQualityLocked("COMPLETION_TIMEOUT")
				}
				r.mu.Unlock()
			}()
		case client.SubmitRateLimit:
			record.HTTPStatus = intPtr(response.HTTPStatus)
			record.HTTPProtocol = response.Protocol
			record.APICode = intPtr(response.APICode)
			if response.RetryAfter != nil {
				milliseconds := response.RetryAfter.Milliseconds()
				record.RetryAfterMS = &milliseconds
			}
			r.poolRateLimit(user.Alias, completed, durationOrZero(response.RetryAfter))
			r.recordFailure(record, model.OutcomeRejected429, "rate_limited", response.Message)
			r.addQuality("UNEXPECTED_429")
			close(done)
		case client.Submit4xx:
			record.HTTPStatus = intPtr(response.HTTPStatus)
			record.HTTPProtocol = response.Protocol
			record.APICode = intPtr(response.APICode)
			if response.HTTPStatus == 401 || response.HTTPStatus == 403 {
				r.poolDisable(user.Alias)
				r.addQuality("AUTH_FAILURE")
			} else {
				r.poolQuarantine(user.Alias, completed)
			}
			r.recordFailure(record, model.OutcomeRejected4xx, "http_4xx", response.Message)
			r.addQuality("HTTP_4XX")
			close(done)
		default:
			record.HTTPStatus = intPtr(response.HTTPStatus)
			record.HTTPProtocol = response.Protocol
			record.APICode = intPtr(response.APICode)
			r.poolQuarantine(user.Alias, completed)
			r.recordFailure(record, model.OutcomeServerError, "http_5xx", response.Message)
			r.addQuality("HTTP_5XX")
			close(done)
		}
	}()
	return record, done
}

// observeRealistic intentionally performs exactly one ticket request and one
// SSE establishment attempt. It does not use observer.Observe because that
// component is intentionally terminal-focused and may reconcile via GET after
// an SSE failure. The realistic admission KPI must not manufacture polling or
// wait for Judge drain.
func (r *benchmarkRun) observeRealistic(parent context.Context, record *model.SubmissionRecord, submissionID int64) {
	api := r.prepared.Clients[record.UserAlias]
	ticketStarted := time.Now()
	r.mu.Lock()
	record.TicketAttempted, record.TicketStartedAt = true, &ticketStarted
	r.activeTickets++
	if r.activeTickets > r.peakTickets {
		r.peakTickets = r.activeTickets
	}
	r.updatePeakLogicalInFlightLocked()
	r.mu.Unlock()
	ticketCtx, cancelTicket := context.WithTimeout(parent, r.prepared.Config.APITimeout)
	ticket, ticketErr := api.IssueTicket(ticketCtx, submissionID)
	cancelTicket()
	ticketCompleted := time.Now()
	r.mu.Lock()
	r.activeTickets--
	record.TicketCompletedAt = &ticketCompleted
	ticketLatency := ticketCompleted.Sub(ticketStarted).Seconds() * 1000
	record.TicketLatencyMS = &ticketLatency
	if ticketErr != nil {
		class := client.TransportErrorClass(ticketErr)
		if class == "ambiguous_post" {
			class = "ticket_error"
		}
		record.Outcome, record.ErrorClass, record.Error = model.OutcomeObserverFailure, class, r.redactor.Sanitize(ticketErr.Error())
		if class == "client_emfile" || class == "client_enfile" || class == "client_ephemeral_port_exhaustion" {
			r.addQualityLocked("LOAD_GENERATOR_LIMITED")
		}
		r.addQualityLocked("TICKET_FAILURE")
		r.mu.Unlock()
		return
	}
	record.TicketSucceeded = true
	r.mu.Unlock()

	// The request deadline leaves a full hold duration after the configured
	// connection budget. There is deliberately no reconnect attempt.
	streamStarted := time.Now()
	streamCtx, cancelStream := context.WithTimeout(parent, r.prepared.Config.SSEConnectTimeout+r.prepared.Config.SSEHoldDuration)
	defer cancelStream()
	r.mu.Lock()
	record.SSEAttempted, record.SSEStartedAt = true, &streamStarted
	r.mu.Unlock()
	response, streamErr := api.OpenEvents(streamCtx, submissionID, ticket.Value)
	if streamErr != nil {
		closed := time.Now()
		r.mu.Lock()
		record.SSEClosedAt, record.SSECloseReason = &closed, "establishment_failed"
		class := client.TransportErrorClass(streamErr)
		if class == "ambiguous_post" {
			class = "sse_establishment_error"
		}
		record.Outcome, record.ErrorClass, record.Error = model.OutcomeObserverFailure, class, r.redactor.Sanitize(streamErr.Error())
		if class == "client_emfile" || class == "client_enfile" || class == "client_ephemeral_port_exhaustion" {
			r.addQualityLocked("LOAD_GENERATOR_LIMITED")
		}
		r.addQualityLocked("SSE_ESTABLISHMENT_FAILURE")
		r.mu.Unlock()
		return
	}
	established := time.Now()
	r.mu.Lock()
	record.SSEEstablished, record.SSEEstablishedAt = true, &established
	establishLatency := established.Sub(streamStarted).Seconds() * 1000
	record.SSEEstablishLatencyMS = &establishLatency
	r.activeSSE++
	if r.activeSSE > r.peakSSE {
		r.peakSSE = r.activeSSE
	}
	r.updatePeakLogicalInFlightLocked()
	r.mu.Unlock()

	terminalStatus, terminal, fullHold, closeReason := holdRealisticSSE(streamCtx, response.Body, r.prepared.Config.SSEHoldDuration)
	closed := time.Now()
	r.mu.Lock()
	r.activeSSE--
	record.SSEClosedAt, record.SSECloseReason = &closed, closeReason
	record.SSESurvivedFullHold, record.SSETerminalDuringHold = fullHold, terminal
	if terminal {
		record.TerminalStatus = terminalStatus
		record.TerminalObservedAt = &closed
		record.CompletionSource = model.CompletionSSEEvent
	}
	r.mu.Unlock()
}

func holdRealisticSSE(ctx context.Context, body io.ReadCloser, hold time.Duration) (status string, terminal, fullHold bool, reason string) {
	read := make(chan sseReadResult, 1)
	go func() {
		value := readSSEUntilTerminal(body)
		read <- value
	}()
	timer := time.NewTimer(hold)
	defer timer.Stop()
	select {
	case value := <-read:
		_ = body.Close()
		if value.terminal {
			return value.status, true, false, "terminal_event"
		}
		if value.err != nil {
			return "", false, false, "stream_error"
		}
		return "", false, false, "stream_closed"
	case <-timer.C:
		_ = body.Close()
		<-read
		return "", false, true, "hold_expired"
	case <-ctx.Done():
		_ = body.Close()
		<-read
		return "", false, false, "stream_context_done"
	}
}

type sseReadResult struct {
	status   string
	terminal bool
	err      error
}

func readSSEUntilTerminal(body io.Reader) sseReadResult {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024), 1<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < len("data:") || line[:len("data:")] != "data:" {
			continue
		}
		var event struct {
			Status string `json:"status"`
		}
		if json.Unmarshal(bytes.TrimSpace([]byte(line[len("data:"):])), &event) == nil && terminalSSEStatus(event.Status) {
			return sseReadResult{status: event.Status, terminal: true}
		}
	}
	return sseReadResult{err: scanner.Err()}
}

func terminalSSEStatus(status string) bool {
	switch status {
	case "PENDING", "JUDGING", "":
		return false
	default:
		return true
	}
}

func (r *benchmarkRun) finish() (*RunResult, error) {
	defer r.limiter.Close()
	defer r.cancel()
	// POSTs scheduled before the load boundary may still be awaiting HTTP
	// responses. They are observed in drain rather than silently discarded.
	r.postWG.Wait()
	r.drainStart = r.loadEnd
	if r.drainStart.IsZero() {
		r.drainStart = time.Now()
	}
	r.metadata.Phases.Drain.StartedAt = r.drainStart.UTC()
	if r.prepared.Config.ObservationMode == config.ObservationFull {
		drainCtx, cancel := context.WithTimeout(r.ctx, r.prepared.Config.DrainTimeout)
		done := make(chan struct{})
		go func() { r.observerWG.Wait(); close(done) }()
		select {
		case <-done:
		case <-drainCtx.Done():
			r.addQuality("DRAIN_TIMEOUT")
			r.cancel()
			<-done
		}
		cancel()
	} else if r.prepared.Config.ObservationMode == config.ObservationRealistic {
		// This waits only for the explicitly bounded ticket/SSE-hold lifecycle;
		// it is not a terminal Judge drain and never issues reconciliation GETs.
		bound := r.prepared.Config.APITimeout + r.prepared.Config.SSEConnectTimeout + r.prepared.Config.SSEHoldDuration + time.Second
		waitCtx, cancel := context.WithTimeout(r.ctx, bound)
		done := make(chan struct{})
		go func() { r.observerWG.Wait(); close(done) }()
		select {
		case <-done:
		case <-waitCtx.Done():
			r.addQuality("REALISTIC_SSE_HOLD_TIMEOUT")
			r.cancel()
			<-done
		}
		cancel()
	}
	r.drainEnd = time.Now()
	r.stopClientSampler()
	drainEnd := r.drainEnd.UTC()
	r.metadata.Phases.Drain.EndedAt = &drainEnd
	r.mu.Lock()
	records := append([]*model.SubmissionRecord(nil), r.records...)
	clientSamples := append([]model.ClientResourceSample(nil), r.clientSamples...)
	r.mu.Unlock()
	if r.parentCtx.Err() != nil {
		r.addQuality("RUN_CANCELLED")
		r.metadata.State = model.RunCancelled
	}
	values := make([]model.SubmissionRecord, 0, len(records))
	for _, record := range records {
		values = append(values, *record)
	}
	var health []model.HealthProbe
	if (r.prepared.Config.ObservationMode == config.ObservationAdmissionOnly || r.prepared.Config.ObservationMode == config.ObservationRealistic) && r.parentCtx.Err() == nil {
		health = r.admissionHealthProbes()
	}
	var scheduleDelays []float64
	for _, record := range values {
		if record.Phase == model.PhaseLoad && record.ScheduleDelayMS != nil {
			scheduleDelays = append(scheduleDelays, *record.ScheduleDelayMS)
		}
	}
	if schedule := stats.Distribution(scheduleDelays); schedule.P95 != nil && time.Duration(*schedule.P95*float64(time.Millisecond)) > r.prepared.Config.ScheduleDelayP95Budget {
		r.addQuality("LOAD_GENERATOR_LIMITED")
	}
	loadPhase := model.PhaseLoad
	loadWindows := stats.BuildWindows(stats.WindowInput{RunID: r.prepared.Config.RunID, Phase: model.PhaseLoad, FilterPhase: &loadPhase, Start: r.loadStart, End: r.loadEnd, Origin: r.loadStart, Window: r.prepared.Config.Window, TargetRate: floatPtr(rateFloat(r.prepared.Config.Rate)), Records: values, PeakInFlight: r.peakInFlight})
	drainWindows := stats.BuildWindows(stats.WindowInput{RunID: r.prepared.Config.RunID, Phase: model.PhaseDrain, FilterPhase: &loadPhase, Start: r.drainStart, End: r.drainEnd, Origin: r.loadStart, Window: r.prepared.Config.Window, Records: values, PeakInFlight: r.peakInFlight})
	windows := append(loadWindows, drainWindows...)
	summary := summarize(r.prepared.Config, values, windows, r.loadStart, r.loadEnd, r.drainStart, r.drainEnd, r.quality, r.peakInFlight)
	if r.prepared.Config.Mode == config.ModeBurst {
		summary.Burst = burstMetrics(values, r.loadStart, r.burstLaunchCompletion, r.peakLogicalInFlight, r.peakObservers)
		// summarize runs before burst metrics are derived from actual event
		// timestamps, so install the canonical pipeline rate here.
		summary.Pipeline.TerminalThroughputPerSec = summary.Burst.PipelineTerminalThroughputPerSec
		summary.Burst.Massive = r.prepared.Config.Objective == config.ObjectiveMassiveBurst
		r.metadata.ClientDiagnostics.GoroutinesBeforeBurst = r.burstGoroutinesBefore
		r.metadata.ClientDiagnostics.GoroutinesAfterLaunch = r.burstGoroutinesAfterLaunch
		r.metadata.ClientDiagnostics.PeakLogicalInFlight = r.peakLogicalInFlight
		r.metadata.ClientDiagnostics.PeakActiveObservers = r.peakObservers
		r.metadata.ClientDiagnostics.PeakActivePosts = r.peakPosts
		r.metadata.ClientDiagnostics.PeakActiveTickets = r.peakTickets
		r.metadata.ClientDiagnostics.PeakActiveSSE = r.peakSSE
	}
	if r.prepared.Config.ObservationMode == config.ObservationAdmissionOnly {
		summary.Admission = admissionMetrics(summary, health, r.quality)
		summary.HealthProbes = health
		summary.Classification = model.Classification(summary.Admission.SystemSurvival)
		summary.ClassificationReasons = []string{"POST_ONLY_OBSERVATION", "TERMINAL_PIPELINE_NOT_COLLECTED"}
	} else if r.prepared.Config.ObservationMode == config.ObservationRealistic {
		summary.Realistic = realisticMetrics(summary, values, health, r.quality, r.peakSSE)
		summary.HealthProbes = health
		summary.Classification = model.Classification(summary.Realistic.SystemSurvival)
		summary.ClassificationReasons = []string{"SUBMIT_TICKET_SSE_HOLD", "NO_GET_RECONCILIATION", "NO_TERMINAL_DRAIN"}
	} else {
		classification, reasons := stats.Classify(stats.ClassificationInput{Mode: model.Mode(r.prepared.Config.Mode), Objective: r.prepared.Config.Objective, TargetRate: rateFloat(r.prepared.Config.Rate), Windows: loadWindows, Summary: summary})
		summary.Classification, summary.ClassificationReasons = classification, reasons
	}
	if r.prepared.TransportDiagnostics != nil {
		diagnostics := r.prepared.TransportDiagnostics.Snapshot()
		r.metadata.ClientDiagnostics.TransportNewConnections = diagnostics.NewConnections
		r.metadata.ClientDiagnostics.TransportReusedConnections = diagnostics.ReusedConnections
		r.metadata.ClientDiagnostics.HTTP1Responses = diagnostics.HTTP1Responses
		r.metadata.ClientDiagnostics.HTTP2Responses = diagnostics.HTTP2Responses
		r.metadata.ClientDiagnostics.OtherProtocolResponses = diagnostics.OtherProtocolResponses
	}
	summary.ClientDiagnostics = r.metadata.ClientDiagnostics
	if r.metadata.State != model.RunAborted && r.metadata.State != model.RunCancelled {
		r.metadata.State = model.RunCompleted
	}
	summary.RunState = r.metadata.State
	ended := time.Now().UTC()
	r.metadata.EndedAt = &ended
	if r.prepared.Config.Mode != config.ModeBurst {
		r.metadata.ObservedRates = &model.ObservedRates{AttemptedPerSecond: summary.Rates.LoadAttemptedPerSecond, AcceptedPerSecond: summary.Rates.LoadAcceptedPerSecond, CompletedPerSecond: summary.Rates.LoadTerminalCompletionSecond, PipelineTerminalPerSecond: summary.Rates.LoadPipelineTerminalCompletionSecond, TerminalSemantics: summary.Rates.TerminalCompletionSemantics}
	}
	if err := r.writer.WriteSubmissions(values); err != nil {
		return &RunResult{Dir: r.writer.Dir, Summary: summary}, fmt.Errorf("write submissions: %w", err)
	}
	if err := r.writer.WriteWindows(windows); err != nil {
		return &RunResult{Dir: r.writer.Dir, Summary: summary}, fmt.Errorf("write windows: %w", err)
	}
	if err := r.writer.WriteClientResources(clientSamples); err != nil {
		return &RunResult{Dir: r.writer.Dir, Summary: summary}, fmt.Errorf("write client resources: %w", err)
	}
	if err := r.writer.WriteSummary(summary); err != nil {
		return &RunResult{Dir: r.writer.Dir, Summary: summary}, fmt.Errorf("write summary: %w", err)
	}
	if err := r.writer.WriteReport(report.Markdown(summary)); err != nil {
		return &RunResult{Dir: r.writer.Dir, Summary: summary}, fmt.Errorf("write report: %w", err)
	}
	if err := r.writer.WriteRun(r.metadata); err != nil {
		return &RunResult{Dir: r.writer.Dir, Summary: summary}, fmt.Errorf("write run metadata: %w", err)
	}
	return &RunResult{Dir: r.writer.Dir, Summary: summary}, nil
}

func (r *benchmarkRun) abort(reason string) {
	r.addQuality(reason)
	r.metadata.State = model.RunAborted
}
func (r *benchmarkRun) addQuality(flag string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.addQualityLocked(flag)
}
func (r *benchmarkRun) addQualityLocked(flag string) {
	r.quality[flag] = struct{}{}
	if r.prepared.Config.ErrorPolicy == config.ErrorPolicyStop {
		r.stopArrivals = true
	}
}
func (r *benchmarkRun) setStarted(record *model.SubmissionRecord, at time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record.Attempted = true
	record.PostStartedAt = &at
	offset := at.Sub(r.loadStart).Milliseconds()
	record.PostStartOffsetMS = &offset
	if record.IntendedAt != nil {
		delay := at.Sub(*record.IntendedAt).Seconds() * 1000
		record.ScheduleDelayMS = &delay
	}
}

func (r *benchmarkRun) updatePeakLogicalInFlightLocked() {
	logical := r.activePosts + r.outstanding + r.activeTickets + r.activeSSE
	if logical > r.peakLogicalInFlight {
		r.peakLogicalInFlight = logical
	}
}
func (r *benchmarkRun) setCompleted(record *model.SubmissionRecord, at time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record.PostCompletedAt = &at
	offset := at.Sub(r.loadStart).Milliseconds()
	record.PostCompletedOffsetMS = &offset
	if record.PostStartedAt != nil {
		latency := at.Sub(*record.PostStartedAt).Seconds() * 1000
		record.SubmitLatencyMS = &latency
	}
}
func (r *benchmarkRun) recordFailure(record *model.SubmissionRecord, outcome model.Outcome, class, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record.Outcome = outcome
	record.ErrorClass = class
	record.Error = r.redactor.Sanitize(message)
	if outcome == model.OutcomeRejected429 {
		record.RateLimited = true
	}
	if r.prepared.Config.ErrorPolicy == config.ErrorPolicyStop && record.Phase == model.PhaseLoad {
		r.stopArrivals = true
	}
}
func (r *benchmarkRun) recordCancelled(phase model.Phase, sequence int, alias string, intended time.Time) {
	record := &model.SubmissionRecord{RunID: r.prepared.Config.RunID, Phase: phase, Sequence: sequence, UserAlias: alias, IntendedAt: &intended, Outcome: model.OutcomeCancelled}
	r.recordCancelledPointer(record)
}
func (r *benchmarkRun) recordCancelledPointer(record *model.SubmissionRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, record)
}
func (r *benchmarkRun) recordMissed(phase model.Phase, sequence int, alias string, intended time.Time) {
	record := &model.SubmissionRecord{RunID: r.prepared.Config.RunID, Phase: phase, Sequence: sequence, UserAlias: alias, IntendedAt: &intended, Outcome: model.OutcomeLocalFailure, ErrorClass: "scheduler_missed_arrival", Error: "arrival missed before POST start"}
	r.mu.Lock()
	r.records = append(r.records, record)
	r.addQualityLocked("LOAD_GENERATOR_LIMITED")
	r.mu.Unlock()
}
func (r *benchmarkRun) arrivalMissed(intended time.Time) bool {
	return time.Since(intended) > r.prepared.Config.ScheduleDelayP95Budget
}
func (r *benchmarkRun) poolAccepted(alias string, at time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_ = r.pool.Accepted(alias, at, r.prepared.Config.SubmitCooldown, r.prepared.Config.CooldownGuard)
}
func (r *benchmarkRun) poolQuarantine(alias string, at time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_ = r.pool.Quarantine(alias, at, r.prepared.Config.SubmitCooldown, r.prepared.Config.CooldownGuard)
}
func (r *benchmarkRun) poolRateLimit(alias string, at time.Time, retry time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_ = r.pool.RateLimited(alias, at, r.prepared.Config.SubmitCooldown, r.prepared.Config.CooldownGuard, retry)
}
func (r *benchmarkRun) poolDisable(alias string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.poolDisableLocked(alias)
}
func (r *benchmarkRun) poolDisableLocked(alias string) {
	_ = r.pool.Disable(alias)
}
func (r *benchmarkRun) countEligible(at time.Time) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.pool.CountEligible(at)
}

func summarize(cfg config.Config, records []model.SubmissionRecord, windows []model.WindowRecord, loadStart, loadEnd, drainStart, drainEnd time.Time, quality map[string]struct{}, peak int) model.RunSummary {
	summary := model.RunSummary{
		SchemaVersion: "astracode.judge-benchmark.summary.v2",
		RunID:         cfg.RunID, RunState: model.RunCompleted, Classification: model.ClassificationInconclusive,
		Verdicts: map[string]int{}, ExternalMetrics: model.ExternalMetrics{}, QualityFlags: sortedFlags(quality),
		Compile:   model.CompileMetrics{IncludedInJudgeCore: false, Availability: "UNAVAILABLE", Reason: "judge-bench artifacts do not record per-submission compile wall timestamps"},
		JudgeCore: model.JudgeCoreMetrics{Definition: "compile_success_to_all_required_testcase_execution_batches_completed", Availability: "UNAVAILABLE", Reason: "judge-bench artifacts do not record compile completion and testcase execution completion timestamps"},
		Pipeline:  model.PipelineMetrics{TerminalThroughputSemantics: "observed terminal completion rate across the submission pipeline; not Judge Core throughput"},
	}
	var submit, e2e, delay []float64
	for _, record := range records {
		if record.Phase != model.PhaseLoad {
			continue
		}
		summary.Counts.Intended++
		if record.Attempted {
			summary.Counts.Attempted++
		}
		if record.Accepted {
			summary.Counts.Accepted++
		}
		if record.TerminalStatus != "" {
			summary.Counts.Terminal++
			summary.Verdicts[record.TerminalStatus]++
			switch record.CompletionSource {
			case model.CompletionSSESnapshot, model.CompletionSSEEvent:
				summary.Observer.SSECompletions++
			}
		}
		summary.Observer.GETReconciliations += record.GETReconciliations
		summary.Observer.SSEFailures += record.SSEFailures
		if record.RateLimited {
			summary.Counts.RateLimited++
		}
		switch record.Outcome {
		case model.OutcomeRejected4xx:
			summary.Counts.Other4xx++
		case model.OutcomeServerError:
			summary.Counts.ServerErrors++
		case model.OutcomeAmbiguousPost:
			summary.Counts.AmbiguousPosts++
			summary.Counts.TransportFailures++
		case model.OutcomeCompletionTimeout:
			summary.Counts.CompletionTimeouts++
		case model.OutcomeUserExhausted:
			summary.Counts.UserPoolExhaustions++
		}
		if record.SubmitLatencyMS != nil {
			submit = append(submit, *record.SubmitLatencyMS)
		}
		if record.EndToEndLatencyMS != nil {
			e2e = append(e2e, *record.EndToEndLatencyMS)
		}
		if record.ScheduleDelayMS != nil {
			delay = append(delay, *record.ScheduleDelayMS)
		}
	}
	duration := loadEnd.Sub(loadStart).Seconds()
	acceptedDuringLoad := 0
	if duration > 0 {
		summary.Rates.TargetArrivalPerSecond = floatPtr(rateFloat(cfg.Rate))
		summary.Rates.LoadAttemptedPerSecond = float64(summary.Counts.Attempted) / duration
		for _, record := range records {
			if record.Phase == model.PhaseLoad && record.Accepted && record.PostCompletedAt != nil && !record.PostCompletedAt.Before(loadStart) && record.PostCompletedAt.Before(loadEnd) {
				acceptedDuringLoad++
			}
		}
		summary.Rates.LoadAcceptedPerSecond = float64(acceptedDuringLoad) / duration
		completed := 0
		for _, r := range records {
			if r.Phase == model.PhaseLoad && r.TerminalObservedAt != nil && !r.TerminalObservedAt.Before(loadStart) && r.TerminalObservedAt.Before(loadEnd) {
				completed++
			}
		}
		summary.Rates.LoadTerminalCompletionSecond = float64(completed) / duration // deprecated compatibility alias
		summary.Rates.LoadPipelineTerminalCompletionSecond = summary.Rates.LoadTerminalCompletionSecond
		summary.Rates.TerminalCompletionSemantics = "observed terminal completion rate during the load window across the pipeline; not Judge Core throughput"
	}
	completedDuringLoad := 0
	for _, record := range records {
		if record.Phase == model.PhaseLoad && record.TerminalObservedAt != nil && !record.TerminalObservedAt.Before(loadStart) && record.TerminalObservedAt.Before(loadEnd) {
			completedDuringLoad++
		}
	}
	postsInFlight := 0
	var starts []time.Time
	for _, record := range records {
		if record.Phase != model.PhaseLoad || record.PostStartedAt == nil {
			continue
		}
		starts = append(starts, *record.PostStartedAt)
		if !record.PostStartedAt.After(loadEnd) && (record.PostCompletedAt == nil || record.PostCompletedAt.After(loadEnd)) {
			postsInFlight++
		}
	}
	boundaryAccepted := 0
	for _, record := range records {
		if record.Phase == model.PhaseLoad && record.Accepted && record.PostCompletedAt != nil && !record.PostCompletedAt.Before(loadEnd) {
			boundaryAccepted++
		}
	}
	loadWindow := model.LoadWindow{DurationMS: loadEnd.Sub(loadStart).Milliseconds(), Accepted: acceptedDuringLoad, BoundaryAcceptedAfterLoad: boundaryAccepted, Completed: completedDuringLoad, OutstandingAtEnd: acceptedDuringLoad - completedDuringLoad, PostsInFlightAtEnd: postsInFlight}
	if cfg.Mode == config.ModeBurst {
		spread := scheduler.BurstSpread(starts).Milliseconds()
		loadWindow.BurstSpreadMS = &spread
	}
	summary.LoadWindow = loadWindow
	summary.Drain = model.Drain{DurationMS: drainEnd.Sub(drainStart).Milliseconds(), OutstandingAtStart: summary.LoadWindow.OutstandingAtEnd, Completed: 0, Remaining: summary.Counts.Accepted - summary.Counts.Terminal, TimedOut: containsFlag(quality, "DRAIN_TIMEOUT")}
	for _, record := range records {
		if record.Phase == model.PhaseLoad && record.TerminalObservedAt != nil && !record.TerminalObservedAt.Before(loadEnd) {
			summary.Drain.Completed++
		}
	}
	if summary.Drain.DurationMS > 0 {
		rate := float64(summary.Drain.Completed) / (float64(summary.Drain.DurationMS) / 1000)
		summary.Drain.CompletionRatePerSecond = &rate
		summary.Drain.PipelineTerminalCompletionRatePerSecond = &rate
	}
	summary.Outstanding = model.Outstanding{Peak: peak, EndOfLoad: summary.LoadWindow.OutstandingAtEnd, EndOfDrain: summary.Drain.Remaining}
	summary.Latencies = model.Latencies{SubmitMS: stats.Distribution(submit), EndToEndMS: stats.Distribution(e2e), ScheduleDelayMS: stats.Distribution(delay)}
	if summary.Counts.Accepted > 0 {
		coverage := float64(summary.Counts.Terminal) / float64(summary.Counts.Accepted)
		summary.Pipeline.TerminalObservationCoverage = &coverage
		summary.Pipeline.RightCensored = summary.Counts.Terminal < summary.Counts.Accepted
	}
	summary.Pipeline.TerminalCompleted = summary.Counts.Terminal
	if cfg.Mode != config.ModeBurst && summary.Rates.LoadPipelineTerminalCompletionSecond > 0 {
		// The standard summary has a load-window terminal rate and a separate
		// drain rate. Keep the canonical top-level value aligned with the legacy
		// load-window field rather than inventing a whole-run terminal rate.
		rate := summary.Rates.LoadPipelineTerminalCompletionSecond
		summary.Pipeline.TerminalThroughputPerSec = &rate
	}
	return summary
}

func burstMetrics(records []model.SubmissionRecord, origin time.Time, launchCompletion *int64, peakLogical, peakObservers int) *model.BurstMetrics {
	starts := make([]time.Time, 0)
	accepted := make([]time.Time, 0)
	terminal := make([]time.Time, 0)
	offsets := make([]float64, 0)
	for _, record := range records {
		if record.Phase != model.PhaseLoad {
			continue
		}
		if record.PostStartedAt != nil {
			starts = append(starts, *record.PostStartedAt)
			offsets = append(offsets, record.PostStartedAt.Sub(origin).Seconds()*1000)
		}
		if record.Accepted && record.PostCompletedAt != nil {
			accepted = append(accepted, *record.PostCompletedAt)
		}
		if record.TerminalObservedAt != nil {
			terminal = append(terminal, *record.TerminalObservedAt)
		}
	}
	attemptedInterval, attemptedRate := intakeMetrics(starts)
	acceptedInterval, acceptedRate := intakeMetrics(accepted)
	terminalInterval, terminalRate := intakeMetrics(terminal)
	return &model.BurstMetrics{AttemptedIntervalMS: attemptedInterval, AcceptedIntervalMS: acceptedInterval, TerminalIntervalMS: terminalInterval, AttemptedThroughputPerSec: attemptedRate, AcceptedThroughputPerSec: acceptedRate, TerminalThroughputPerSec: terminalRate, PipelineTerminalIntervalMS: terminalInterval, PipelineTerminalThroughputPerSec: terminalRate, TerminalThroughputSemantics: "observed terminal completion rate across the pipeline; not Judge Core throughput", PostStartOffsetMS: stats.Distribution(offsets), LaunchCompletionMS: launchCompletion, PeakLogicalInFlight: peakLogical, PeakActiveObservers: peakObservers}
}

// intakeMetrics uses the span of observed client timestamps. A single event
// has no measurable intake interval and is therefore unavailable, never zero.
func intakeMetrics(values []time.Time) (*int64, *float64) {
	if len(values) < 2 {
		return nil, nil
	}
	min, max := values[0], values[0]
	for _, value := range values[1:] {
		if value.Before(min) {
			min = value
		}
		if value.After(max) {
			max = value
		}
	}
	span := max.Sub(min)
	if span <= 0 {
		return nil, nil
	}
	milliseconds := span.Milliseconds()
	rate := float64(len(values)) / span.Seconds()
	return &milliseconds, &rate
}

func runMetadata(cfg config.Config, sourceSHA string, sourceBytes int64) model.RunMetadata {
	now := time.Now().UTC()
	sha, dirty := gitInfo()
	return model.RunMetadata{SchemaVersion: "astracode.judge-benchmark.run.v1", BenchmarkVersion: model.BenchmarkVersion, RunID: cfg.RunID, Mode: model.Mode(cfg.Mode), Repetition: cfg.Repetition, Seed: cfg.Seed, State: model.RunCompleted, StartedAt: now, Repository: model.Repository{GitSHA: sha, Dirty: dirty}, Target: model.Target{BaseURL: cfg.BaseURL.Scheme + "://" + cfg.BaseURL.Host, ProblemID: cfg.ProblemID, ProblemSlug: cfg.ProblemSlug, Language: cfg.Language, ExpectedVerdict: cfg.ExpectedVerdict, SourceSHA256: sourceSHA, SourceBytes: sourceBytes}, Users: model.UserSet{Configured: 0, Selected: 0}, Workload: model.Workload{WindowMS: cfg.Window.Milliseconds(), WarmupCount: cfg.WarmupCount, SubmitCooldownMS: cfg.SubmitCooldown.Milliseconds(), CooldownGuardMS: cfg.CooldownGuard.Milliseconds(), SubmitLatencyBudgetMS: cfg.SubmitLatencyBudget.Milliseconds(), PoolHeadroomPercent: cfg.PoolHeadroomPercent, MaxSubmissions: cfg.MaxSubmissions, MaxInFlight: cfg.MaxInFlight}, Timeouts: model.Timeouts{APIMS: cfg.APITimeout.Milliseconds(), SubmissionMS: cfg.SubmissionTimeout.Milliseconds(), DrainMS: cfg.DrainTimeout.Milliseconds()}, Observer: model.ObserverConfig{SSEPrimary: true, ConnectMS: cfg.SSEConnectTimeout.Milliseconds(), HoldMS: cfg.SSEHoldDuration.Milliseconds(), IdleMS: cfg.SSEIdleTimeout.Milliseconds(), MaxReconnects: cfg.SSEMaxReconnects, BackoffBaseMS: cfg.SSEBackoffBase.Milliseconds(), BackoffMaxMS: cfg.SSEBackoffMax.Milliseconds(), SafetyReconcileIntervalMS: cfg.SafetyReconcileInterval.Milliseconds(), ReconcileMaxQPS: cfg.ReconcileMaxQPS}, SystemConfig: cfg.SystemConfig}
}
func generatedRunID(cfg config.Config) string {
	prefix := "B"
	if cfg.Mode == config.ModeSustained {
		prefix = "S"
	}
	return fmt.Sprintf("%s-R%d-%s", prefix, cfg.Repetition, time.Now().UTC().Format("20060102T150405Z"))
}
func requiredSessionHorizon(cfg config.Config) time.Duration {
	return requiredSessionHorizonFor(cfg, 1)
}

func requiredSessionHorizonFor(cfg config.Config, sessions int) time.Duration {
	if cfg.Mode == config.ModeSustained {
		load := cfg.Duration
		if cfg.TotalSubmissions > 0 && cfg.Rate != nil {
			load = scheduler.OffsetFor(cfg.Rate, cfg.TotalSubmissions-1)
		}
		return cfg.WarmupTimeout + load + cfg.DrainTimeout + cfg.AuthValidityMargin
	}
	// The SSE endpoint authenticates the established stream with its short-lived
	// ticket, not the access cookie. Access validity must therefore cover
	// pre-run warmup plus the bounded burst POST/ticket-start phase, rather than
	// an arbitrarily long Judge drain. Reconnect/reconciliation after expiry is
	// still recorded as an observer limitation rather than hidden.
	if sessions < 1 {
		sessions = 1
	}
	concurrency := cfg.PreflightConcurrency
	if concurrency < 1 {
		concurrency = 1
	}
	batches := (sessions + concurrency - 1) / concurrency
	// Each batch may consume the bounded API timeout. This is deliberately a
	// conservative access-token planning horizon, not a claim about server
	// performance. It keeps a 10k preflight from validating an early session
	// that expires before the common burst release.
	// A refresh-required/needed-auto session can perform refresh plus /me in
	// one worker slot. Use the two-request upper bound, not an optimistic
	// single GET estimate, when proving the token survives to burst release.
	preflight := saturatingDurationMultiply(cfg.APITimeout, batches*2)
	warmup := time.Duration(0)
	if cfg.WarmupCount > 0 {
		warmup = cfg.WarmupTimeout
	}
	return preflight + warmup + cfg.BurstStartTimeout + cfg.AuthValidityMargin
}

func saturatingDurationMultiply(value time.Duration, factor int) time.Duration {
	if value <= 0 || factor <= 0 {
		return 0
	}
	max := int64(^uint64(0) >> 1)
	if int64(factor) > max/int64(value) {
		return time.Duration(max)
	}
	return value * time.Duration(factor)
}

func prepareAccessSession(ctx context.Context, cfg config.Config, session *credentials.Session, api *client.API) error {
	return prepareAccessSessionFor(ctx, cfg, 1, session, api)
}

func prepareAccessSessionFor(ctx context.Context, cfg config.Config, sessions int, session *credentials.Session, api *client.API) error {
	return ensureAccessSessionFor(ctx, cfg, sessions, session, api, true)
}

// prepareAccessSessionFinalFor honors required-mode's initial refresh without
// issuing a needless second refresh immediately before the run. It refreshes
// only if the remaining token can no longer cover the same bounded horizon.
func prepareAccessSessionFinalFor(ctx context.Context, cfg config.Config, sessions int, session *credentials.Session, api *client.API) error {
	return ensureAccessSessionFor(ctx, cfg, sessions, session, api, false)
}

func ensureAccessSessionFor(ctx context.Context, cfg config.Config, sessions int, session *credentials.Session, api *client.API, forceRequiredRefresh bool) error {
	coversHorizon := accessCoversHorizonFor(session.AccessToken(cfg.BaseURL), cfg, sessions)
	switch cfg.RefreshMode {
	case config.RefreshRequired:
		if !session.HasRefresh(cfg.BaseURL) {
			return errors.New("refresh is required but no refresh token is available")
		}
		if forceRequiredRefresh || !coversHorizon {
			if err := refreshWithTimeout(ctx, api, cfg.APITimeout); err != nil {
				return fmt.Errorf("refresh access session: %w", err)
			}
		}
	case config.RefreshAuto:
		if !coversHorizon {
			if !session.HasRefresh(cfg.BaseURL) {
				return errors.New("access token cannot cover benchmark horizon and no refresh token is available")
			}
			if err := refreshWithTimeout(ctx, api, cfg.APITimeout); err != nil {
				return fmt.Errorf("refresh access session: %w", err)
			}
		}
	case config.RefreshOff:
		if !coversHorizon {
			return errors.New("access token cannot cover benchmark horizon while --session-refresh=off")
		}
	default:
		return errors.New("invalid session refresh mode")
	}
	if !accessCoversHorizonFor(session.AccessToken(cfg.BaseURL), cfg, sessions) {
		return errors.New("access token cannot cover benchmark horizon after refresh")
	}
	return nil
}

func accessCoversHorizon(token string, cfg config.Config) bool {
	return accessCoversHorizonFor(token, cfg, 1)
}

func accessCoversHorizonFor(token string, cfg config.Config, sessions int) bool {
	expiresAt, ok := tokenExpiry(token)
	return ok && time.Until(expiresAt) >= requiredSessionHorizonFor(cfg, sessions)
}
func tokenExpiry(token string) (time.Time, bool) {
	parts := splitJWT(token)
	if len(parts) != 3 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, false
	}
	var claim struct {
		Exp json.Number `json:"exp"`
	}
	dec := json.NewDecoder(bytesReader(payload))
	dec.UseNumber()
	if dec.Decode(&claim) != nil {
		return time.Time{}, false
	}
	value, err := claim.Exp.Int64()
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(value, 0), true
}
func splitJWT(value string) []string         { return stringsSplit(value, '.') }
func bytesReader(value []byte) *bytes.Reader { return bytes.NewReader(value) }
func stringsSplit(value string, separator byte) []string {
	parts := []string{}
	start := 0
	for i := 0; i < len(value); i++ {
		if value[i] == separator {
			parts = append(parts, value[start:i])
			start = i + 1
		}
	}
	return append(parts, value[start:])
}
func meWithTimeout(ctx context.Context, api *client.API, timeout time.Duration) (client.Me, error) {
	request, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return api.Me(request)
}

func (r *benchmarkRun) admissionHealthProbes() []model.HealthProbe {
	if len(r.prepared.validationIndexes) == 0 {
		return nil
	}
	indexes := append([]int(nil), r.prepared.validationIndexes...)
	if len(indexes) > 4 {
		indexes = indexes[:4]
	}
	probes := make([]model.HealthProbe, 0, len(indexes)+1)
	start := time.Now()
	_, err := publicProblemWithTimeout(r.parentCtx, r.prepared.Clients[r.prepared.Sessions[indexes[0]].Alias], r.prepared.Config.APITimeout, r.prepared.Config.ProblemSlug)
	probes = append(probes, healthProbe("public_problem", start, err))
	for _, index := range indexes {
		start = time.Now()
		me, meErr := meWithTimeout(r.parentCtx, r.prepared.Clients[r.prepared.Sessions[index].Alias], r.prepared.Config.APITimeout)
		if meErr == nil && (!me.IsActive || me.Role != "user" || me.ID == "") {
			meErr = errors.New("unexpected authenticated identity state")
		}
		probes = append(probes, healthProbe("authenticated_me", start, meErr))
	}
	return probes
}

func healthProbe(name string, start time.Time, err error) model.HealthProbe {
	probe := model.HealthProbe{Name: name, At: start.UTC(), LatencyMS: float64(time.Since(start)) / float64(time.Millisecond)}
	if err == nil {
		probe.Status = "PASS"
		return probe
	}
	probe.Status, probe.ErrorClass = "FAIL", "request_error"
	return probe
}

func admissionMetrics(summary model.RunSummary, probes []model.HealthProbe, quality map[string]struct{}) *model.AdmissionMetrics {
	metrics := &model.AdmissionMetrics{ObservationMode: string(config.ObservationAdmissionOnly), ClientQualification: "QUALIFIED", ExternalSurvivalEvidence: "UNAVAILABLE"}
	if summary.Burst != nil {
		metrics.EffectiveAcceptedIntakePerSec = summary.Burst.AcceptedThroughputPerSec
		metrics.PostStartSpreadMS = summary.LoadWindow.BurstSpreadMS
	}
	for _, probe := range probes {
		if probe.Status != "PASS" {
			metrics.SystemSurvival = string(model.ClassificationFailed)
			return metrics
		}
	}
	if containsFlag(quality, "LOAD_GENERATOR_LIMITED") || containsFlag(quality, "MAX_IN_FLIGHT_REACHED") {
		metrics.ClientQualification = "CLIENT_LIMITED"
		metrics.SystemSurvival = string(model.ClassificationClientLimited)
		return metrics
	}
	if summary.Counts.Attempted != summary.Counts.Intended || summary.Counts.ServerErrors > 0 || summary.Counts.TransportFailures > 0 || summary.Counts.AmbiguousPosts > 0 {
		metrics.SystemSurvival = string(model.ClassificationFailed)
		return metrics
	}
	if summary.Counts.Accepted != summary.Counts.Intended || summary.Counts.RateLimited > 0 || summary.Counts.Other4xx > 0 {
		metrics.SystemSurvival = string(model.ClassificationDegradedSurvival)
		return metrics
	}
	// Container restart/health evidence is collected externally. Do not claim a
	// clean system-survival result before that evidence has been correlated.
	metrics.SystemSurvival = string(model.ClassificationDegradedSurvival)
	return metrics
}

func realisticMetrics(summary model.RunSummary, records []model.SubmissionRecord, probes []model.HealthProbe, quality map[string]struct{}, peakSSE int) *model.RealisticMetrics {
	metrics := &model.RealisticMetrics{ObservationMode: string(config.ObservationRealistic), ClientQualification: "QUALIFIED", ExternalSurvivalEvidence: "UNAVAILABLE"}
	metrics.Submission.Attempted, metrics.Submission.Successful = summary.Counts.Attempted, summary.Counts.Accepted
	metrics.Submission.SuccessPercent = ratio(metrics.Submission.Successful, metrics.Submission.Attempted)
	metrics.Submission.LatencyMS = summary.Latencies.SubmitMS
	if summary.Burst != nil {
		metrics.Submission.ThroughputPerSec = summary.Burst.AcceptedThroughputPerSec
	}
	var ticketLatencies, sseLatencies []float64
	var ticketStarts, sseEstablished, sseStarts []time.Time
	metrics.SSE.PeakActiveStreams, metrics.SSE.CloseReasons = peakSSE, map[string]int{}
	for _, record := range records {
		if record.Phase != model.PhaseLoad {
			continue
		}
		if record.TicketAttempted {
			metrics.Ticket.Attempted++
			if record.TicketStartedAt != nil {
				ticketStarts = append(ticketStarts, *record.TicketStartedAt)
			}
		}
		if record.TicketSucceeded {
			metrics.Ticket.Successful++
		}
		if record.TicketLatencyMS != nil {
			ticketLatencies = append(ticketLatencies, *record.TicketLatencyMS)
		}
		if record.SSEAttempted {
			metrics.SSE.Attempted++
			if record.SSEStartedAt != nil {
				sseStarts = append(sseStarts, *record.SSEStartedAt)
			}
		}
		if record.SSEEstablished {
			metrics.SSE.Established++
			if record.SSEEstablishedAt != nil {
				sseEstablished = append(sseEstablished, *record.SSEEstablishedAt)
			}
		}
		if record.SSEEstablishLatencyMS != nil {
			sseLatencies = append(sseLatencies, *record.SSEEstablishLatencyMS)
		}
		if record.SSECloseReason != "" {
			metrics.SSE.CloseReasons[record.SSECloseReason]++
		}
		if record.SSESurvivedFullHold {
			metrics.SSE.SurvivedFullHold++
		}
		if record.SSEEstablished && !record.SSESurvivedFullHold && !record.SSETerminalDuringHold {
			metrics.SSE.ClosedEarly++
		}
		if record.SSETerminalDuringHold {
			metrics.SSE.TerminalDuringHold++
		}
	}
	metrics.Ticket.SuccessPercent = ratio(metrics.Ticket.Successful, metrics.Ticket.Attempted)
	metrics.Ticket.LatencyMS = stats.Distribution(ticketLatencies)
	_, metrics.Ticket.ThroughputPerSec = intakeMetrics(ticketStarts)
	metrics.SSE.Failed = metrics.SSE.Attempted - metrics.SSE.Established
	metrics.SSE.EstablishmentPercent = ratio(metrics.SSE.Established, metrics.SSE.Attempted)
	metrics.SSE.EstablishmentLatencyMS = stats.Distribution(sseLatencies)
	_, metrics.SSE.EstablishmentRatePerSec = intakeMetrics(sseEstablished)
	metrics.SSE.StartSpreadMS, _ = intakeMetrics(sseStarts)
	metrics.FullFlowSuccessPercent = ratio(metrics.SSE.Established, summary.Counts.Intended)
	for _, probe := range probes {
		if probe.Status != "PASS" {
			metrics.SystemSurvival = string(model.ClassificationFailed)
			return metrics
		}
	}
	if containsFlag(quality, "LOAD_GENERATOR_LIMITED") || containsFlag(quality, "MAX_IN_FLIGHT_REACHED") {
		metrics.ClientQualification = "CLIENT_LIMITED"
		metrics.SystemSurvival = string(model.ClassificationClientLimited)
		return metrics
	}
	if summary.Counts.Attempted != summary.Counts.Intended || summary.Counts.ServerErrors > 0 || summary.Counts.TransportFailures > 0 || summary.Counts.AmbiguousPosts > 0 || metrics.Ticket.Successful != summary.Counts.Accepted || metrics.SSE.Established != metrics.Ticket.Successful {
		metrics.SystemSurvival = string(model.ClassificationFailed)
		return metrics
	}
	if summary.Counts.Accepted != summary.Counts.Intended || summary.Counts.RateLimited > 0 || summary.Counts.Other4xx > 0 || metrics.SSE.ClosedEarly > 0 {
		metrics.SystemSurvival = string(model.ClassificationDegradedSurvival)
		return metrics
	}
	// Container/restart evidence is external. Keep the conservative result
	// degraded until offline collector correlation can demonstrate clean survival.
	metrics.SystemSurvival = string(model.ClassificationDegradedSurvival)
	return metrics
}

func ratio(numerator, denominator int) *float64 {
	if denominator == 0 {
		return nil
	}
	value := float64(numerator) / float64(denominator)
	return &value
}
func publicProblemWithTimeout(ctx context.Context, api *client.API, timeout time.Duration, slug string) (client.Problem, error) {
	request, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return api.PublicProblem(request, slug)
}
func refreshWithTimeout(ctx context.Context, api *client.API, timeout time.Duration) error {
	request, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return api.Refresh(request)
}
func checkResultRoot(root string) error {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	probe, err := os.MkdirTemp(root, ".judge-bench-preflight-")
	if err != nil {
		return err
	}
	return os.Remove(probe)
}
func warmupP95(records []*model.SubmissionRecord) time.Duration {
	values := []float64{}
	for _, record := range records {
		if record.Phase == model.PhaseWarmup && record.SubmitLatencyMS != nil {
			values = append(values, *record.SubmitLatencyMS)
		}
	}
	d := stats.Distribution(values)
	if d.P95 == nil {
		return 0
	}
	return time.Duration(*d.P95 * float64(time.Millisecond))
}
func sleepContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
func sleepUntil(ctx context.Context, at time.Time) bool {
	select {
	case <-ctx.Done():
		return false
	default:
	}
	d := time.Until(at)
	if d <= 0 {
		return true
	}
	return sleepContext(ctx, d)
}
func durationOrZero(value *time.Duration) time.Duration {
	if value == nil {
		return 0
	}
	return *value
}
func rateFloat(value *big.Rat) float64 {
	if value == nil {
		return 0
	}
	f, _ := value.Float64()
	return f
}
func floatPtr(value float64) *float64 { return &value }
func intPtr(value int) *int           { return &value }
func sortedFlags(values map[string]struct{}) []string {
	flags := make([]string, 0, len(values))
	for value := range values {
		flags = append(flags, value)
	}
	sort.Strings(flags)
	return flags
}
func containsFlag(values map[string]struct{}, target string) bool { _, ok := values[target]; return ok }
func gitInfo() (string, bool) {
	sha := "unknown"
	if output, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		sha = string(bytes.TrimSpace(output))
	}
	dirty := exec.Command("git", "diff", "--quiet").Run() != nil
	return sha, dirty
}
