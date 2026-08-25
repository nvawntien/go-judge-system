// Package runner coordinates preflight, optional pre-warmup refresh, warmup,
// open-loop load, SSE-first completion, drain, and result persistence.
package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"sort"
	"sync"
	"time"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/client"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/config"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/credentials"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/model"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/observer"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/report"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/result"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/scheduler"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/stats"
)

type Prepared struct {
	Config        config.Config
	Sessions      []*credentials.Session
	Clients       map[string]*client.API
	Source        []byte
	SourceSHA256  string
	Subjects      map[string]string // alias -> real ID; memory-only
	RequiredUsers int
}

// maxSourceBytes mirrors the public submission contract (256 KiB) without
// importing a production service internal package.
const maxSourceBytes = 256 * 1024

// Preflight performs only filesystem checks and GET requests. It never creates
// a submission, ticket, refresh, Redis key, or server-side configuration state.
func Preflight(ctx context.Context, cfg config.Config) (*Prepared, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	file, err := credentials.Load(cfg.UsersFile)
	if err != nil {
		return nil, err
	}
	sessions, err := credentials.NewSessions(file, cfg.BaseURL, nil)
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
	prepared := &Prepared{Config: cfg, Sessions: sessions, Clients: make(map[string]*client.API, len(sessions)), Source: source, SourceSHA256: fmt.Sprintf("%x", hash[:]), Subjects: make(map[string]string, len(sessions))}
	subjects := map[string]string{}
	for _, session := range sessions {
		api, err := client.New(cfg.BaseURL, session)
		if err != nil {
			return nil, err
		}
		me, err := meWithTimeout(ctx, api, cfg.APITimeout)
		if err != nil {
			return nil, fmt.Errorf("preflight session %q: %w", session.Alias, err)
		}
		if me.ID == "" || !me.IsActive || me.Role != "user" {
			return nil, fmt.Errorf("benchmark session %q is not an active normal user", session.Alias)
		}
		if _, duplicate := subjects[me.ID]; duplicate {
			return nil, errors.New("benchmark sessions resolve to duplicate authenticated identities")
		}
		subjects[me.ID] = session.Alias
		session.Subject = me.ID
		prepared.Subjects[session.Alias] = me.ID
		prepared.Clients[session.Alias] = api
	}
	problem, err := publicProblemWithTimeout(ctx, prepared.Clients[sessions[0].Alias], cfg.APITimeout, cfg.ProblemSlug)
	if err != nil {
		return nil, fmt.Errorf("preflight public problem: %w", err)
	}
	if problem.ID != cfg.ProblemID {
		return nil, fmt.Errorf("problem slug resolved to ID %d, want %d", problem.ID, cfg.ProblemID)
	}
	if cfg.Mode == config.ModeSustained {
		required, err := scheduler.RequiredUsers(cfg.Rate, cfg.SubmitCooldown, cfg.CooldownGuard, cfg.SubmitLatencyBudget, cfg.PoolHeadroomPercent)
		if err != nil {
			return nil, err
		}
		prepared.RequiredUsers = required
		if len(sessions) < required {
			return nil, fmt.Errorf("benchmark user pool has %d users, need at least %d", len(sessions), required)
		}
	} else if len(sessions) < cfg.BurstSize {
		return nil, fmt.Errorf("benchmark user pool has %d users, need %d distinct burst users", len(sessions), cfg.BurstSize)
	}
	if err := checkResultRoot(cfg.ResultRoot); err != nil {
		return nil, err
	}
	return prepared, nil
}

// PrepareSessions may refresh only before warmup. It does not write rotated
// cookies to disk and requires the post-refresh access token to cover the run.
func PrepareSessions(ctx context.Context, prepared *Prepared) error {
	cfg := prepared.Config
	for _, session := range prepared.Sessions {
		api := prepared.Clients[session.Alias]
		switch cfg.RefreshMode {
		case config.RefreshRequired:
			if !session.HasRefresh(cfg.BaseURL) {
				return fmt.Errorf("session %q has no refresh token", session.Alias)
			}
			if err := refreshWithTimeout(ctx, api, cfg.APITimeout); err != nil {
				return fmt.Errorf("refresh session %q: %w", session.Alias, err)
			}
		case config.RefreshAuto:
			if session.HasRefresh(cfg.BaseURL) {
				if err := refreshWithTimeout(ctx, api, cfg.APITimeout); err != nil {
					return fmt.Errorf("refresh session %q: %w", session.Alias, err)
				}
			}
		case config.RefreshOff:
			// Explicitly frozen access session.
		}
		me, err := meWithTimeout(ctx, api, cfg.APITimeout)
		if err != nil || me.ID != prepared.Subjects[session.Alias] || !me.IsActive || me.Role != "user" {
			return fmt.Errorf("prepared session %q no longer has the preflight identity", session.Alias)
		}
		expiresAt, ok := tokenExpiry(session.AccessToken(cfg.BaseURL))
		if !ok || time.Until(expiresAt) < requiredSessionHorizon(cfg) {
			return fmt.Errorf("session %q access lifetime cannot cover warmup, load, drain, and margin", session.Alias)
		}
	}
	return nil
}

type RunResult struct {
	Dir     string
	Summary model.RunSummary
}

func Run(ctx context.Context, prepared *Prepared) (*RunResult, error) {
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
	run.metadata.Users = model.UserSet{Configured: len(prepared.Sessions), Selected: len(prepared.Sessions)}
	if cfg.Mode == config.ModeBurst {
		burstSize, jitter := cfg.BurstSize, cfg.Jitter.Milliseconds()
		run.metadata.Workload.BurstSize = &burstSize
		run.metadata.Workload.JitterMilliseconds = &jitter
	} else {
		rate, duration := rateFloat(cfg.Rate), cfg.Duration.Milliseconds()
		run.metadata.Workload.TargetRatePerSecond = &rate
		run.metadata.Workload.ArrivalDurationMS = &duration
	}
	if err := writer.WriteRun(run.metadata); err != nil {
		return nil, err
	}
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
	parentCtx    context.Context
	ctx          context.Context
	cancel       context.CancelFunc
	prepared     *Prepared
	writer       *result.Writer
	pool         *scheduler.Pool
	limiter      *observer.ReconcileLimiter
	redactor     result.Redactor
	metadata     model.RunMetadata
	mu           sync.Mutex
	records      []*model.SubmissionRecord
	acceptedIDs  map[int64]struct{}
	quality      map[string]struct{}
	postWG       sync.WaitGroup
	observerWG   sync.WaitGroup
	activePosts  int
	outstanding  int
	peakInFlight int
	stopArrivals bool
	loadStart    time.Time
	loadEnd      time.Time
	drainStart   time.Time
	drainEnd     time.Time
}

func newRun(ctx context.Context, prepared *Prepared, writer *result.Writer) *benchmarkRun {
	aliases := make([]string, 0, len(prepared.Sessions))
	for _, session := range prepared.Sessions {
		aliases = append(aliases, session.Alias)
	}
	pool, _ := scheduler.NewPool(aliases, uint64(prepared.Config.Seed))
	runContext, cancel := context.WithCancel(ctx)
	secrets := []string{string(prepared.Source)}
	for _, session := range prepared.Sessions {
		for _, cookie := range session.Jar.Cookies(prepared.Config.BaseURL) {
			secrets = append(secrets, cookie.Value)
		}
	}
	for _, subject := range prepared.Subjects {
		secrets = append(secrets, subject)
	}
	return &benchmarkRun{parentCtx: ctx, ctx: runContext, cancel: cancel, prepared: prepared, writer: writer, pool: pool, limiter: observer.NewReconcileLimiter(prepared.Config.ReconcileMaxQPS), redactor: result.NewRedactor(secrets...), quality: map[string]struct{}{}, acceptedIDs: map[int64]struct{}{}}
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
	r.metadata.Phases.Load.StartedAt = r.loadStart.UTC()
	for index, arrival := range plan {
		intended := r.loadStart.Add(arrival.Offset)
		if !sleepUntil(r.ctx, intended) {
			r.recordCancelled(model.PhaseLoad, index, arrival.Alias, intended)
			continue
		}
		if r.arrivalMissed(intended) {
			r.recordMissed(model.PhaseLoad, index, arrival.Alias, intended)
			continue
		}
		r.launch(r.ctx, model.PhaseLoad, index, intended, arrival.Alias)
	}
	r.loadEnd = time.Now()
	loadEnd := r.loadEnd.UTC()
	r.metadata.Phases.Load.EndedAt = &loadEnd
}

func (r *benchmarkRun) sustained() {
	cfg := r.prepared.Config
	r.loadStart = time.Now()
	r.metadata.Phases.Load.StartedAt = r.loadStart.UTC()
	count, err := scheduler.ArrivalCount(cfg.Rate, cfg.Duration)
	if err != nil {
		r.addQuality("SCHEDULER_PLAN_FAILED")
		count = 0
	}
	for index := 0; index < count; index++ {
		offset := config.OffsetFor(cfg.Rate, index)
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
	// Arrival generation ends exactly at the configured boundary, independently
	// of request or Judge completion.
	boundary := r.loadStart.Add(cfg.Duration)
	sleepUntil(r.ctx, boundary)
	r.loadEnd = boundary
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
	r.postWG.Add(1)
	r.mu.Unlock()
	go func() {
		defer r.postWG.Done()
		started := time.Now()
		r.setStarted(record, started)
		requestCtx, cancel := context.WithTimeout(ctx, r.prepared.Config.APITimeout)
		response, err := r.prepared.Clients[user.Alias].Submit(requestCtx, client.SubmissionRequest{ProblemID: r.prepared.Config.ProblemID, Language: r.prepared.Config.Language, SourceCode: string(r.prepared.Source)})
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
			r.recordFailure(record, model.OutcomeAmbiguousPost, "ambiguous_post", err.Error())
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
			record.APICode = intPtr(response.APICode)
			record.SubmissionID = &id
			record.InitialStatus = response.Submission.Status
			r.outstanding++
			if r.outstanding > r.peakInFlight {
				r.peakInFlight = r.outstanding
			}
			r.mu.Unlock()
			r.poolAccepted(user.Alias, completed)
			r.observerWG.Add(1)
			go func() {
				defer r.observerWG.Done()
				defer close(done)
				observerStarted := time.Now()
				r.mu.Lock()
				record.ObserverStartedAt = &observerStarted
				r.mu.Unlock()
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
			record.APICode = intPtr(response.APICode)
			r.poolQuarantine(user.Alias, completed)
			r.recordFailure(record, model.OutcomeServerError, "http_5xx", response.Message)
			r.addQuality("HTTP_5XX")
			close(done)
		}
	}()
	return record, done
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
	r.drainEnd = time.Now()
	drainEnd := r.drainEnd.UTC()
	r.metadata.Phases.Drain.EndedAt = &drainEnd
	r.mu.Lock()
	records := append([]*model.SubmissionRecord(nil), r.records...)
	r.mu.Unlock()
	if r.parentCtx.Err() != nil {
		r.addQuality("RUN_CANCELLED")
		r.metadata.State = model.RunCancelled
	}
	values := make([]model.SubmissionRecord, 0, len(records))
	for _, record := range records {
		values = append(values, *record)
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
	classification, reasons := stats.Classify(stats.ClassificationInput{Mode: model.Mode(r.prepared.Config.Mode), Objective: r.prepared.Config.Objective, TargetRate: rateFloat(r.prepared.Config.Rate), Windows: loadWindows, Summary: summary})
	summary.Classification, summary.ClassificationReasons = classification, reasons
	if r.metadata.State != model.RunAborted && r.metadata.State != model.RunCancelled {
		r.metadata.State = model.RunCompleted
	}
	summary.RunState = r.metadata.State
	ended := time.Now().UTC()
	r.metadata.EndedAt = &ended
	r.metadata.ObservedRates = &model.ObservedRates{AttemptedPerSecond: summary.Rates.LoadAttemptedPerSecond, AcceptedPerSecond: summary.Rates.LoadAcceptedPerSecond, CompletedPerSecond: summary.Rates.LoadTerminalCompletionSecond}
	if err := r.writer.WriteSubmissions(values); err != nil {
		return &RunResult{Dir: r.writer.Dir, Summary: summary}, fmt.Errorf("write submissions: %w", err)
	}
	if err := r.writer.WriteWindows(windows); err != nil {
		return &RunResult{Dir: r.writer.Dir, Summary: summary}, fmt.Errorf("write windows: %w", err)
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
	summary := model.RunSummary{SchemaVersion: "astracode.judge-benchmark.summary.v1", RunID: cfg.RunID, RunState: model.RunCompleted, Classification: model.ClassificationInconclusive, Verdicts: map[string]int{}, ExternalMetrics: model.ExternalMetrics{}, QualityFlags: sortedFlags(quality)}
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
		}
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
		summary.Rates.LoadTerminalCompletionSecond = float64(completed) / duration
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
	}
	summary.Outstanding = model.Outstanding{Peak: peak, EndOfLoad: summary.LoadWindow.OutstandingAtEnd, EndOfDrain: summary.Drain.Remaining}
	summary.Latencies = model.Latencies{SubmitMS: stats.Distribution(submit), EndToEndMS: stats.Distribution(e2e), ScheduleDelayMS: stats.Distribution(delay)}
	return summary
}

func runMetadata(cfg config.Config, sourceSHA string, sourceBytes int64) model.RunMetadata {
	now := time.Now().UTC()
	sha, dirty := gitInfo()
	return model.RunMetadata{SchemaVersion: "astracode.judge-benchmark.run.v1", BenchmarkVersion: model.BenchmarkVersion, RunID: cfg.RunID, Mode: model.Mode(cfg.Mode), Repetition: cfg.Repetition, Seed: cfg.Seed, State: model.RunCompleted, StartedAt: now, Repository: model.Repository{GitSHA: sha, Dirty: dirty}, Target: model.Target{BaseURL: cfg.BaseURL.Scheme + "://" + cfg.BaseURL.Host, ProblemID: cfg.ProblemID, ProblemSlug: cfg.ProblemSlug, Language: cfg.Language, ExpectedVerdict: cfg.ExpectedVerdict, SourceSHA256: sourceSHA, SourceBytes: sourceBytes}, Users: model.UserSet{Configured: 0, Selected: 0}, Workload: model.Workload{WindowMS: cfg.Window.Milliseconds(), WarmupCount: cfg.WarmupCount, SubmitCooldownMS: cfg.SubmitCooldown.Milliseconds(), CooldownGuardMS: cfg.CooldownGuard.Milliseconds(), SubmitLatencyBudgetMS: cfg.SubmitLatencyBudget.Milliseconds(), PoolHeadroomPercent: cfg.PoolHeadroomPercent, MaxSubmissions: cfg.MaxSubmissions, MaxInFlight: cfg.MaxInFlight}, Timeouts: model.Timeouts{APIMS: cfg.APITimeout.Milliseconds(), SubmissionMS: cfg.SubmissionTimeout.Milliseconds(), DrainMS: cfg.DrainTimeout.Milliseconds()}, Observer: model.ObserverConfig{SSEPrimary: true, ConnectMS: cfg.SSEConnectTimeout.Milliseconds(), IdleMS: cfg.SSEIdleTimeout.Milliseconds(), MaxReconnects: cfg.SSEMaxReconnects, BackoffBaseMS: cfg.SSEBackoffBase.Milliseconds(), BackoffMaxMS: cfg.SSEBackoffMax.Milliseconds(), SafetyReconcileIntervalMS: cfg.SafetyReconcileInterval.Milliseconds(), ReconcileMaxQPS: cfg.ReconcileMaxQPS}}
}
func generatedRunID(cfg config.Config) string {
	prefix := "B"
	if cfg.Mode == config.ModeSustained {
		prefix = "S"
	}
	return fmt.Sprintf("%s-R%d-%s", prefix, cfg.Repetition, time.Now().UTC().Format("20060102T150405Z"))
}
func requiredSessionHorizon(cfg config.Config) time.Duration {
	if cfg.Mode == config.ModeSustained {
		return cfg.WarmupTimeout + cfg.Duration + cfg.DrainTimeout + cfg.AuthValidityMargin
	}
	return cfg.WarmupTimeout + cfg.SubmissionTimeout + cfg.AuthValidityMargin
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
