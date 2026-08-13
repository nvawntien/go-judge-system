'use client';

import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Header } from '@/components/Header';
import { useAuth } from '@/components/AuthProvider';
import { useToast } from '@/components/ToastProvider';
import { CodeEditor } from '@/components/CodeEditor';
import {
  ProblemNotFound,
  ProblemPanel,
  ProblemPanelSkeleton,
  type ProblemTab,
} from '@/components/workspace/ProblemPanel';
import { Icon, Spinner } from '@/components/ui';
import { ApiError, NetworkError, problemApi, submissionApi } from '@/lib/api';
import {
  LANGUAGES,
  formatDateTime,
  formatMemoryKb,
  formatRuntimeMs,
  formatTestcaseCount,
  isPendingStatus,
  isTerminalSubmissionStatus,
  languageMeta,
  verdictMeta,
} from '@/lib/format';
import { useDismissable, useOnline, useViewportWidth } from '@/lib/hooks';
import { fetchProgress, invalidateProgress } from '@/lib/progress';
import { useSubmissionStream, type SubmissionStreamState } from '@/lib/submission-stream';
import { CODE_TEMPLATES, draftKey } from '@/lib/templates';
import type {
  CodeDiagnostic,
  LanguageCode,
  Problem,
  RunResponse,
  RunTestCaseResult,
  Submission,
  SubmissionDetail,
  SubmissionStreamEvent,
} from '@/lib/types';

type BottomTab = 'tests' | 'console' | 'result';
type LoadState = 'loading' | 'ready' | 'notfound' | 'error';

interface TestCase {
  id: string;
  kind: 'sample' | 'custom';
  name: string;
  input: string;
  expected: string | null;
  /** Examples come from the problem and cannot be removed. */
  fromExample: boolean;
}

export default function WorkspacePage() {
  const params = useParams<{ slug: string }>();
  const slug = params?.slug ?? '';
  const router = useRouter();
  const { user, loading: authLoading } = useAuth();
  const { showToast } = useToast();
  const online = useOnline();
  const width = useViewportWidth();

  const stacked = width < 1020;
  const isMobile = width < 760;
  const [mobileView, setMobileView] = useState<'problem' | 'editor'>('problem');

  /* -------------------------------------------------------------- problem */

  const [problem, setProblem] = useState<Problem | null>(null);
  const [loadState, setLoadState] = useState<LoadState>('loading');
  const [loadError, setLoadError] = useState<string>('');
  const [tab, setTab] = useState<ProblemTab>('description');

  useEffect(() => {
    if (!slug) return;
    const controller = new AbortController();
    setLoadState('loading');

    problemApi
      .bySlug(slug, controller.signal)
      .then((res) => {
        setProblem(res);
        setLoadState('ready');
      })
      .catch((err) => {
        if (controller.signal.aborted) return;
        if (err instanceof ApiError && err.isNotFound) {
          setLoadState('notfound');
        } else {
          setLoadError(
            err instanceof NetworkError
              ? 'AstraCode is temporarily unreachable. Check your connection and try again.'
              : 'This problem could not be loaded.',
          );
          setLoadState('error');
        }
      });

    return () => controller.abort();
  }, [slug]);

  /* ------------------------------------------------------------- progress */

  const [solved, setSolved] = useState(false);
  const [attempted, setAttempted] = useState(false);

  const syncProgress = useCallback(
    async (force = false) => {
      if (!user || !problem) return;
      try {
        const progress = await fetchProgress(force);
        setSolved(progress.solvedIds.has(problem.id));
        setAttempted(progress.attemptedIds.has(problem.id));
      } catch {
        /* progress is decorative */
      }
    },
    [user, problem],
  );

  useEffect(() => {
    void syncProgress();
  }, [syncProgress]);

  /* --------------------------------------------------------------- editor */

  const [language, setLanguage] = useState<LanguageCode>('GO');
  const [code, setCode] = useState<string>(CODE_TEMPLATES.GO);
  const [fontSize, setFontSize] = useState(13);
  const [tabSize, setTabSize] = useState(4);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [resetDialog, setResetDialog] = useState(false);
  const [draftRestored, setDraftRestored] = useState(false);
  const [draftSavedAt, setDraftSavedAt] = useState<number | null>(null);
  const [jumpLine, setJumpLine] = useState<number | null>(null);
  const [codeDiagnostics, setCodeDiagnostics] = useState<CodeDiagnostic[]>([]);
  const [fullscreen, setFullscreen] = useState(false);
  const [restoreSubmission, setRestoreSubmission] = useState<SubmissionDetail | null>(null);
  const [historyRefreshKey, setHistoryRefreshKey] = useState(0);

  const settingsRef = useDismissable<HTMLDivElement>(settingsOpen, () => setSettingsOpen(false));

  // Editor preferences are global; the language is remembered too.
  useEffect(() => {
    try {
      const storedLang = localStorage.getItem('astra-lang');
      if (storedLang && LANGUAGES.some((l) => l.code === storedLang)) {
        setLanguage(storedLang as LanguageCode);
      }
      const storedFont = Number(localStorage.getItem('astra-font'));
      if (storedFont >= 11 && storedFont <= 18) setFontSize(storedFont);
      const storedTab = Number(localStorage.getItem('astra-tab'));
      if ([2, 4, 8].includes(storedTab)) setTabSize(storedTab);
    } catch {
      /* ignore */
    }
  }, []);

  // Load the draft (or template) whenever the problem/language pair changes.
  useEffect(() => {
    if (!problem) return;
    setCodeDiagnostics([]);
    setJumpLine(null);
    let restored = false;
    try {
      const draft = localStorage.getItem(draftKey(problem.id, language));
      if (draft) {
        setCode(draft);
        restored = true;
      }
    } catch {
      /* ignore */
    }
    if (!restored) setCode(CODE_TEMPLATES[language]);
    setDraftRestored(restored);
    setDraftSavedAt(null);
  }, [problem, language]);

  // Debounced draft persistence.
  useEffect(() => {
    if (!problem) return;
    const id = setTimeout(() => {
      try {
        localStorage.setItem(draftKey(problem.id, language), code);
        setDraftSavedAt(Date.now());
      } catch {
        /* quota exceeded — the editor still works, just without a draft */
      }
    }, 600);
    return () => clearTimeout(id);
  }, [code, problem, language]);

  const changeLanguage = (next: LanguageCode) => {
    setLanguage(next);
    setCodeDiagnostics([]);
    setJumpLine(null);
    try {
      localStorage.setItem('astra-lang', next);
    } catch {
      /* ignore */
    }
  };

  const restoreSubmissionCode = useCallback(
    (detail: SubmissionDetail) => {
      if (!problem) return;
      const nextLanguage = detail.language.toUpperCase() as LanguageCode;
      if (!LANGUAGES.some((item) => item.code === nextLanguage)) {
        showToast(`Cannot restore unsupported language: ${detail.language}`, 'error');
        return;
      }

      try {
        localStorage.setItem('astra-lang', nextLanguage);
        localStorage.setItem(draftKey(problem.id, nextLanguage), detail.source_code);
      } catch {
        /* editor still updates even if local draft persistence is unavailable */
      }

      setLanguage(nextLanguage);
      setCode(detail.source_code);
      setCodeDiagnostics([]);
      setDraftRestored(false);
      setDraftSavedAt(Date.now());
      setJumpLine(null);
      setRestoreSubmission(null);
      setBottomOpen(true);
      setBottomTab('tests');
      if (stacked) setMobileView('editor');
      showToast(`Submission #${detail.id} restored to the editor`, 'success');
    },
    [problem, showToast, stacked],
  );

  const requestRestoreSubmissionCode = useCallback(
    (detail: SubmissionDetail) => {
      const nextLanguage = detail.language.toUpperCase() as LanguageCode;
      const willReplaceDraft = code !== detail.source_code || language !== nextLanguage;
      if (willReplaceDraft) {
        setRestoreSubmission(detail);
        return;
      }
      restoreSubmissionCode(detail);
    },
    [code, language, restoreSubmissionCode],
  );

  /* ---------------------------------------------------------------- tests */

  const [tests, setTests] = useState<TestCase[]>([]);
  const [selectedTest, setSelectedTest] = useState(0);

  useEffect(() => {
    if (!problem) return;
    const fromExamples = (problem.examples ?? []).map((example, index) => ({
      id: `sample-${index + 1}`,
      kind: 'sample' as const,
      name: `Case ${index + 1}`,
      input: example.input,
      expected: example.expected_output,
      fromExample: true,
    }));
    setTests(
      fromExamples.length > 0
        ? fromExamples
        : [{ id: 'custom-1', kind: 'custom', name: 'Custom 1', input: '', expected: null, fromExample: false }],
    );
    setSelectedTest(0);
  }, [problem]);

  const updateTest = (index: number, patch: Partial<TestCase>) => {
    setTests((current) => current.map((test, i) => (i === index ? { ...test, ...patch } : test)));
  };

  const addTest = () => {
    if (tests.length >= 10) {
      showToast('Maximum of 10 test cases', 'error');
      return;
    }
    const customCount = tests.filter((test) => test.kind === 'custom').length + 1;
    setTests((current) => [
      ...current,
      {
        id: `custom-${Date.now()}`,
        kind: 'custom',
        name: `Custom ${customCount}`,
        input: '',
        expected: null,
        fromExample: false,
      },
    ]);
    setSelectedTest(tests.length);
  };

  const removeTest = (index: number) => {
    setTests((current) => current.filter((_, i) => i !== index));
    setSelectedTest((current) => Math.max(0, current > index ? current - 1 : current));
  };

  /* --------------------------------------------------------- run & submit */

  const [bottomTab, setBottomTab] = useState<BottomTab>('tests');
  const [bottomOpen, setBottomOpen] = useState(true);
  const [running, setRunning] = useState(false);
  const [runResult, setRunResult] = useState<RunResponse | null>(null);
  const [consoleLines, setConsoleLines] = useState<{ text: string; color: string }[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [checkingResult, setCheckingResult] = useState(false);
  const [submission, setSubmission] = useState<Submission | null>(null);
  const [activeSubmissionId, setActiveSubmissionId] = useState<number | null>(null);
  const [liveUpdateError, setLiveUpdateError] = useState('');
  const [detailFetchError, setDetailFetchError] = useState('');
  const terminalDetailHandledRef = useRef(false);
  const terminalDetailAbortRef = useRef<AbortController | null>(null);

  useEffect(
    () => () => {
      terminalDetailAbortRef.current?.abort();
    },
    [],
  );

  useEffect(() => {
    terminalDetailAbortRef.current?.abort();
    terminalDetailHandledRef.current = false;
    setActiveSubmissionId(null);
    setSubmitting(false);
    setCheckingResult(false);
    setSubmission(null);
    setLiveUpdateError('');
    setDetailFetchError('');
  }, [slug]);

  useEffect(() => {
    if (user) return;
    terminalDetailAbortRef.current?.abort();
    terminalDetailHandledRef.current = false;
    setActiveSubmissionId(null);
    setSubmitting(false);
    setCheckingResult(false);
  }, [user]);

  const requireAuth = (action: string): boolean => {
    if (user) return true;
    showToast(`Sign in to ${action}`, 'error');
    router.push(`/login?next=/problems/${slug}`);
    return false;
  };

  const run = async () => {
    if (!problem || running || submitting) return;
    if (!requireAuth('run code')) return;
    if (!online) {
      showToast('You appear to be offline', 'error');
      return;
    }

    setRunning(true);
    setCodeDiagnostics([]);
    setJumpLine(null);
    setBottomTab('result');

    try {
      const result = await submissionApi.run({
        problem_id: problem.id,
        language,
        source_code: code,
        testcases: tests.map((test) => ({
          id: test.id,
          kind: test.kind,
          stdin: test.input,
          expected_output: test.expected,
        })),
      });
      setRunResult(result);
      const diagnostics = collectRunDiagnostics(result);
      setCodeDiagnostics(diagnostics);
      if (diagnostics[0]?.line) setJumpLine(diagnostics[0].line);
      const firstFailed = result.tests.findIndex((test) => isRunFailure(test));
      if (firstFailed >= 0) setSelectedTest(firstFailed);
      else if (selectedTest >= tests.length) setSelectedTest(0);
      setConsoleLines(buildRunConsoleLines(result));
    } catch (err) {
      if (err instanceof ApiError && err.isUnimplemented) {
        showToast('Custom runs are unavailable right now — use Submit', 'error');
        setConsoleLines([
          {
            text: `POST ${process.env.NEXT_PUBLIC_RUN_ENDPOINT ?? '/api/v1/submissions/run'} → ${err.httpStatus}`,
            color: 'var(--text3)',
          },
          {
            text: 'Custom tests are unavailable right now. Submit runs the official test suite.',
            color: 'var(--warn)',
          },
        ]);
        setBottomTab('console');
      } else if (err instanceof NetworkError) {
        showToast('AstraCode is temporarily unreachable. Check your connection and try again.', 'error');
      } else if (err instanceof ApiError) {
        showToast(err.message || 'Run failed', 'error');
      }
    } finally {
      setRunning(false);
    }
  };

  const applyStreamStatus = useCallback((event: SubmissionStreamEvent) => {
    setSubmission((current) => {
      if (!current || current.id !== event.submission_id) return current;
      return {
        ...current,
        status: event.status,
        updated_at: event.updated_at,
      };
    });
  }, []);

  const finishSubmissionFromDetail = useCallback(
    (detail: Submission) => {
      setSubmission(detail);
      setSubmitting(false);
      setActiveSubmissionId(null);
      setLiveUpdateError('');
      setDetailFetchError('');
      setHistoryRefreshKey((current) => current + 1);
      const verdict = verdictMeta(detail.status);
      showToast(
        detail.status === 'ACCEPTED' ? 'Accepted' : `Verdict: ${verdict.label}`,
        detail.status === 'ACCEPTED' ? 'success' : 'error',
      );
      invalidateProgress();
      void syncProgress(true);
    },
    [showToast, syncProgress],
  );

  const fetchTerminalSubmissionDetail = useCallback(
    async (id: number, terminalEvent?: SubmissionStreamEvent) => {
      if (terminalDetailHandledRef.current) return;
      terminalDetailHandledRef.current = true;
      terminalDetailAbortRef.current?.abort();
      const controller = new AbortController();
      terminalDetailAbortRef.current = controller;

      if (terminalEvent) {
        setSubmission((current) => {
          if (!current || current.id !== terminalEvent.submission_id) return current;
          return {
            ...current,
            status: terminalEvent.status,
            updated_at: terminalEvent.updated_at,
          };
        });
      }

      try {
        const detail = await submissionApi.get(id, controller.signal);
        if (controller.signal.aborted) return;
        finishSubmissionFromDetail(detail);
      } catch (err) {
        if (controller.signal.aborted) return;
        setSubmitting(false);
        setActiveSubmissionId(null);
        setDetailFetchError(
          err instanceof NetworkError
            ? 'Verdict received, but submission details could not be loaded. Check your connection and try again.'
            : err instanceof ApiError
              ? err.message || 'Verdict received, but submission detail could not be loaded.'
              : 'Verdict received, but submission detail could not be loaded.',
        );
      } finally {
        if (terminalDetailAbortRef.current === controller) {
          terminalDetailAbortRef.current = null;
        }
      }
    },
    [finishSubmissionFromDetail],
  );

  const checkResultOnce = useCallback(async () => {
    if (!submission || checkingResult) return;
    setCheckingResult(true);
    setDetailFetchError('');
    try {
      const latest = await submissionApi.get(submission.id);
      setSubmission(latest);
      if (isTerminalSubmissionStatus(latest.status)) {
        finishSubmissionFromDetail(latest);
      } else {
        showToast(`Submission #${latest.id} is still ${verdictMeta(latest.status).label.toLowerCase()}.`, 'success');
      }
    } catch (err) {
      if (err instanceof NetworkError) {
        setDetailFetchError('AstraCode is temporarily unreachable. Check your connection and try again.');
      } else if (err instanceof ApiError) {
        setDetailFetchError(err.message || 'Could not read the submission.');
      } else {
        setDetailFetchError('Could not read the submission.');
      }
    } finally {
      setCheckingResult(false);
    }
  }, [checkingResult, finishSubmissionFromDetail, showToast, submission]);

  const handleStreamTerminal = useCallback(
    (event: SubmissionStreamEvent) => {
      applyStreamStatus(event);
      void fetchTerminalSubmissionDetail(event.submission_id, event);
    },
    [applyStreamStatus, fetchTerminalSubmissionDetail],
  );

  const handleStreamError = useCallback((error: { message: string }) => {
    setLiveUpdateError(error.message);
    setSubmitting(false);
  }, []);

  const stream = useSubmissionStream({
    submissionId: activeSubmissionId,
    enabled: Boolean(activeSubmissionId && submission && isPendingStatus(submission.status)),
    onStatus: applyStreamStatus,
    onTerminal: handleStreamTerminal,
    onError: handleStreamError,
  });

  const submit = async () => {
    if (!problem || submitting || running) return;
    if (!requireAuth('submit')) return;
    if (!online) {
      showToast('You appear to be offline', 'error');
      return;
    }
    if (!code.trim()) {
      showToast('Write some code before submitting', 'error');
      return;
    }

    setSubmitting(true);
    terminalDetailAbortRef.current?.abort();
    terminalDetailHandledRef.current = false;
    setActiveSubmissionId(null);
    setLiveUpdateError('');
    setDetailFetchError('');
    setSubmission(null);
    setRunResult(null);
    setCodeDiagnostics([]);
    setJumpLine(null);
    setBottomTab('result');
    setBottomOpen(true);

    try {
      const created = await submissionApi.create({
        problem_id: problem.id,
        language,
        source_code: code,
      });
      setSubmission({
        id: created.id,
        problem_id: created.problem_id,
        problem_title: created.problem_title,
        user_id: user?.id ?? '',
        username: user?.username ?? '',
        language: created.language,
        source_code: code,
        status: created.status,
        execution_time_ms: null,
        memory_used_kb: null,
        passed_testcases: null,
        total_testcases: null,
        compile_output: null,
        error_message: null,
        created_at: created.created_at,
        updated_at: created.created_at,
      });
      if (isTerminalSubmissionStatus(created.status)) {
        void fetchTerminalSubmissionDetail(created.id);
      } else {
        setActiveSubmissionId(created.id);
      }
    } catch (err) {
      setSubmitting(false);
      if (err instanceof NetworkError) {
        showToast('AstraCode is temporarily unreachable. Check your connection and try again.', 'error');
      } else if (err instanceof ApiError) {
        showToast(err.message || 'Submission rejected', 'error');
      } else {
        showToast('Submission failed', 'error');
      }
    }
  };

  // Ctrl/Cmd+Enter runs, Ctrl/Cmd+Shift+Enter submits.
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (!(event.ctrlKey || event.metaKey) || event.key !== 'Enter') return;
      event.preventDefault();
      if (event.shiftKey) void submit();
      else void run();
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  });

  /* ---------------------------------------------------------------- split */

  const [splitPct, setSplitPct] = useState(52);
  const [bottomPx, setBottomPx] = useState(235);
  const draggingRef = useRef<'h' | 'v' | null>(null);
  const shellRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    try {
      const stored = Number(sessionStorage.getItem('astra-split'));
      if (stored >= 30 && stored <= 68) setSplitPct(stored);
      const storedBottom = Number(sessionStorage.getItem('astra-bottom'));
      if (storedBottom >= 120 && storedBottom <= 520) setBottomPx(storedBottom);
    } catch {
      /* ignore */
    }
  }, []);

  useEffect(() => {
    const onMove = (event: MouseEvent) => {
      const mode = draggingRef.current;
      if (!mode || !shellRef.current) return;
      const rect = shellRef.current.getBoundingClientRect();

      if (mode === 'h') {
        const pct = Math.min(68, Math.max(30, ((event.clientX - rect.left) / rect.width) * 100));
        setSplitPct(pct);
      } else {
        const px = Math.min(520, Math.max(120, rect.bottom - event.clientY));
        setBottomPx(px);
      }
    };

    const onUp = () => {
      if (!draggingRef.current) return;
      draggingRef.current = null;
      document.body.style.userSelect = '';
      try {
        sessionStorage.setItem('astra-split', String(splitPct));
        sessionStorage.setItem('astra-bottom', String(bottomPx));
      } catch {
        /* ignore */
      }
    };

    window.addEventListener('mousemove', onMove);
    window.addEventListener('mouseup', onUp);
    return () => {
      window.removeEventListener('mousemove', onMove);
      window.removeEventListener('mouseup', onUp);
    };
  }, [splitPct, bottomPx]);

  const startDrag = (mode: 'h' | 'v') => (event: React.MouseEvent) => {
    event.preventDefault();
    draggingRef.current = mode;
    document.body.style.userSelect = 'none';
  };

  /* -------------------------------------------------------------- derived */

  const langMeta = languageMeta(language);
  const busy = running || submitting || checkingResult;
  const runResultsByID = useMemo(() => {
    const entries = runResult?.tests.map((test) => [test.id, test] as const) ?? [];
    return new Map(entries);
  }, [runResult]);

  const bottomTabs: { key: BottomTab; label: string }[] = useMemo(
    () => [
      { key: 'tests', label: `Tests (${tests.length})` },
      { key: 'console', label: 'Console' },
      { key: 'result', label: 'Result' },
    ],
    [tests.length],
  );

  const leftVisible = !fullscreen && (!stacked || mobileView === 'problem');
  const rightVisible = !stacked || mobileView === 'editor' || fullscreen;

  /* -------------------------------------------------------------- render */

  if (loadState === 'loading' || (authLoading && !problem)) {
    return (
      <div style={{ minHeight: '100vh', background: 'var(--bg)' }}>
        <Header />
        <ProblemPanelSkeleton />
      </div>
    );
  }

  if (loadState === 'notfound') {
    return (
      <div style={{ minHeight: '100vh', background: 'var(--bg)' }}>
        <Header />
        <ProblemNotFound slug={slug} />
      </div>
    );
  }

  if (loadState === 'error' || !problem) {
    return (
      <div style={{ minHeight: '100vh', background: 'var(--bg)' }}>
        <Header />
        <div style={{ padding: '64px 24px', textAlign: 'center' }}>
          <p style={{ margin: '0 0 6px', fontSize: 14, fontWeight: 600 }}>Couldn&apos;t load the problem</p>
          <p style={{ margin: '0 0 16px', fontSize: 12.5, color: 'var(--text3)', fontFamily: 'var(--font-mono)' }}>
            {loadError}
          </p>
          <Link href="/problems" style={{ fontSize: 13, fontWeight: 600 }}>
            ← Back to problems
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div style={{ minHeight: '100vh', background: 'var(--bg)', display: 'flex', flexDirection: 'column' }}>
      <Header />

      <main
        style={{
          height: 'calc(100vh - 57px)',
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
          animation: 'acFadeUp .25s ease',
        }}
      >
        {stacked && (
          <div
            role="tablist"
            aria-label="Workspace views"
            style={{
              display: 'flex',
              gap: 4,
              padding: '6px 10px',
              borderBottom: '1px solid var(--border)',
              background: 'var(--surface)',
              flexShrink: 0,
            }}
          >
            {(['problem', 'editor'] as const).map((view) => {
              const active = mobileView === view;
              return (
                <button
                  key={view}
                  type="button"
                  role="tab"
                  aria-selected={active}
                  onClick={() => setMobileView(view)}
                  style={{
                    flex: 1,
                    minHeight: 44,
                    border: 'none',
                    borderRadius: 8,
                    background: active ? 'var(--accent-soft)' : 'transparent',
                    color: active ? 'var(--accent-fg)' : 'var(--text2)',
                    fontSize: 13,
                    fontWeight: active ? 600 : 500,
                    cursor: 'pointer',
                  }}
                >
                  {view === 'problem' ? 'Problem' : 'Code'}
                </button>
              );
            })}
          </div>
        )}

        <div ref={shellRef} style={{ flex: 1, display: 'flex', minHeight: 0 }}>
          {leftVisible && (
            <div
              style={{
                width: stacked ? '100%' : `${splitPct}%`,
                height: '100%',
                display: 'flex',
                flexDirection: 'column',
                minWidth: 0,
                background: 'var(--surface)',
                borderRight: stacked ? 'none' : '1px solid var(--border)',
                overflow: 'hidden',
              }}
            >
              <ProblemPanel
                problem={problem}
                tab={tab}
                onTabChange={setTab}
                solved={solved}
                attempted={attempted}
                signedIn={Boolean(user)}
                historyRefreshKey={historyRefreshKey}
                onUseExample={(input, expected) => {
                  setTests((current) => {
                    const next = [
                      ...current,
                      {
                        id: `custom-${Date.now()}`,
                        kind: 'custom' as const,
                        name: `Custom ${current.filter((test) => test.kind === 'custom').length + 1}`,
                        input,
                        expected,
                        fromExample: false,
                      },
                    ];
                    setSelectedTest(next.length - 1);
                    return next;
                  });
                  setBottomTab('tests');
                  setBottomOpen(true);
                  showToast('Example added as a custom test', 'success');
                  if (stacked) setMobileView('editor');
                }}
                onUseSubmissionCode={requestRestoreSubmissionCode}
              />
            </div>
          )}

          {!stacked && !fullscreen && (
            <div
              onMouseDown={startDrag('h')}
              onDoubleClick={() => setSplitPct(52)}
              role="separator"
              aria-orientation="vertical"
              aria-label="Resize panels — drag, or double-click to reset"
              tabIndex={0}
              title="Drag to resize · double-click to reset"
              className="ac-hover-accent-soft2"
              style={{
                width: 8,
                cursor: 'col-resize',
                background: 'var(--surface2)',
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                gap: 3,
                flexShrink: 0,
                transition: 'background .12s',
              }}
            >
              {[0, 1, 2].map((dot) => (
                <span
                  key={dot}
                  style={{ width: 3, height: 3, borderRadius: '50%', background: 'var(--border2)' }}
                />
              ))}
            </div>
          )}

          {rightVisible && (
            <div
              style={{
                width: stacked || fullscreen ? '100%' : `${100 - splitPct}%`,
                height: '100%',
                display: 'flex',
                flexDirection: 'column',
                minWidth: 0,
                background: 'var(--code-bg)',
                overflow: 'hidden',
              }}
            >
              {!online && (
                <div
                  role="alert"
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 9,
                    padding: '8px 14px',
                    background: 'var(--warn-bg)',
                    borderBottom: '1px solid var(--warn)',
                    flexShrink: 0,
                  }}
                >
                  <span
                    aria-hidden="true"
                    style={{ width: 7, height: 7, borderRadius: '50%', background: 'var(--warn)' }}
                  />
                  <span style={{ fontSize: 12, fontWeight: 550 }}>
                    Network disconnected — code is saved locally; Run and Submit are disabled until you
                    reconnect.
                  </span>
                </div>
              )}

              {draftRestored && (
                <div
                  role="status"
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 9,
                    padding: '8px 14px',
                    background: 'var(--accent-soft)',
                    borderBottom: '1px solid var(--accent-soft2)',
                    flexShrink: 0,
                  }}
                >
                  <span
                    aria-hidden="true"
                    style={{ width: 7, height: 7, borderRadius: '50%', background: 'var(--accent)' }}
                  />
                  <span style={{ fontSize: 12, fontWeight: 550, flex: 1 }}>
                    Draft restored from this device.
                  </span>
                  <button
                    type="button"
                    onClick={() => setDraftRestored(false)}
                    aria-label="Dismiss"
                    className="ac-hover-text"
                    style={{
                      border: 'none',
                      background: 'none',
                      color: 'var(--text3)',
                      fontSize: 14,
                      cursor: 'pointer',
                      padding: '2px 6px',
                    }}
                  >
                    ✕
                  </button>
                </div>
              )}

              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  padding: '8px 12px',
                  borderBottom: '1px solid var(--border)',
                  background: 'var(--surface)',
                  flexShrink: 0,
                }}
              >
                <select
                  value={language}
                  onChange={(event) => changeLanguage(event.target.value as LanguageCode)}
                  aria-label="Programming language"
                  style={{
                    height: 32,
                    borderRadius: 8,
                    border: '1px solid var(--border)',
                    background: 'var(--surface2)',
                    padding: '0 8px',
                    fontSize: 12.5,
                    fontWeight: 600,
                    color: 'var(--text)',
                    cursor: 'pointer',
                    fontFamily: 'var(--font-mono)',
                  }}
                >
                  {LANGUAGES.map((item) => (
                    <option key={item.code} value={item.code}>
                      {item.label}
                    </option>
                  ))}
                </select>
                <span style={{ fontSize: 11, color: 'var(--text3)', fontFamily: 'var(--font-mono)' }}>
                  main.{langMeta.ext}
                </span>
                {draftSavedAt !== null && (
                  <span
                    style={{ fontSize: 10.5, color: 'var(--text3)', fontFamily: 'var(--font-mono)' }}
                  >
                    draft saved
                  </span>
                )}

                <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: 6 }}>
                  <div style={{ position: 'relative' }} ref={settingsRef}>
                    <button
                      type="button"
                      onClick={() => setSettingsOpen(!settingsOpen)}
                      aria-label="Editor settings"
                      aria-expanded={settingsOpen}
                      title="Editor settings — font size, tab size"
                      className="ac-hover-surface2"
                      style={iconButton}
                    >
                      <Icon.Gear />
                    </button>
                    {settingsOpen && (
                      <div
                        role="menu"
                        aria-label="Editor settings"
                        style={{
                          position: 'absolute',
                          right: 0,
                          top: 38,
                          width: 210,
                          background: 'var(--surface)',
                          border: '1px solid var(--border)',
                          borderRadius: 12,
                          boxShadow: 'var(--shadow-lg)',
                          padding: 12,
                          zIndex: 30,
                          animation: 'acPop .15s ease',
                        }}
                      >
                        <label style={settingsLabel}>Font size — {fontSize}px</label>
                        <input
                          type="range"
                          min={11}
                          max={18}
                          value={fontSize}
                          onChange={(event) => {
                            const next = Number(event.target.value);
                            setFontSize(next);
                            try {
                              localStorage.setItem('astra-font', String(next));
                            } catch {
                              /* ignore */
                            }
                          }}
                          aria-label="Editor font size"
                          style={{ width: '100%', accentColor: 'var(--accent)' }}
                        />
                        <label style={{ ...settingsLabel, margin: '12px 0 6px' }}>Tab size</label>
                        <select
                          value={tabSize}
                          onChange={(event) => {
                            const next = Number(event.target.value);
                            setTabSize(next);
                            try {
                              localStorage.setItem('astra-tab', String(next));
                            } catch {
                              /* ignore */
                            }
                          }}
                          aria-label="Tab size"
                          style={{
                            width: '100%',
                            height: 32,
                            borderRadius: 8,
                            border: '1px solid var(--border)',
                            background: 'var(--surface2)',
                            padding: '0 8px',
                            fontSize: 12.5,
                            color: 'var(--text)',
                          }}
                        >
                          <option value={2}>2 spaces</option>
                          <option value={4}>4 spaces</option>
                          <option value={8}>8 spaces</option>
                        </select>
                      </div>
                    )}
                  </div>

                  <button
                    type="button"
                    onClick={() => setResetDialog(true)}
                    aria-label="Reset code to template"
                    title="Reset code"
                    className="ac-hover-surface2"
                    style={iconButton}
                  >
                    <Icon.Reset />
                  </button>

                  {!stacked && (
                    <button
                      type="button"
                      onClick={() => setFullscreen(!fullscreen)}
                      aria-label={fullscreen ? 'Exit full-width editor' : 'Full-width editor'}
                      aria-pressed={fullscreen}
                      title={fullscreen ? 'Exit full-width editor' : 'Full-width editor'}
                      className="ac-hover-surface2"
                      style={iconButton}
                    >
                      {fullscreen ? <Icon.Collapse /> : <Icon.Expand />}
                    </button>
                  )}
                </div>
              </div>

              <CodeEditor
                value={code}
                onChange={(next) => {
                  setCode(next);
                  setCodeDiagnostics([]);
                  setJumpLine(null);
                }}
                language={language}
                fontSize={fontSize}
                tabSize={tabSize}
                highlightLine={jumpLine}
                diagnostics={codeDiagnostics}
              />

              <div
                onMouseDown={startDrag('v')}
                onDoubleClick={() => setBottomPx(235)}
                role="separator"
                aria-orientation="horizontal"
                aria-label="Resize result panel — drag, or double-click to reset"
                tabIndex={0}
                className="ac-hover-accent-soft2"
                style={{
                  height: 8,
                  cursor: 'row-resize',
                  background: 'var(--surface2)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  flexShrink: 0,
                  borderTop: '1px solid var(--border)',
                }}
              >
                <span
                  style={{ width: 34, height: 3, borderRadius: 2, background: 'var(--border2)' }}
                />
              </div>

              <div
                style={{
                  flexShrink: 0,
                  height: bottomOpen ? bottomPx : 39,
                  borderTop: '1px solid var(--border)',
                  background: 'var(--surface)',
                  display: 'flex',
                  flexDirection: 'column',
                  maxHeight: '60%',
                  minHeight: 0,
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 2,
                    padding: '0 12px',
                    borderBottom: '1px solid var(--border)',
                    overflowX: 'auto',
                    flexShrink: 0,
                  }}
                >
                  {bottomTabs.map((item) => {
                    const active = bottomTab === item.key;
                    return (
                      <button
                        key={item.key}
                        type="button"
                        role="tab"
                        aria-selected={active}
                        onClick={() => {
                          setBottomTab(item.key);
                          setBottomOpen(true);
                        }}
                        className="ac-hover-text"
                        style={{
                          height: 38,
                          padding: '0 12px',
                          border: 'none',
                          borderBottom: `2px solid ${active ? 'var(--accent)' : 'transparent'}`,
                          background: 'none',
                          cursor: 'pointer',
                          fontSize: 12,
                          fontWeight: 600,
                          color: active ? 'var(--text)' : 'var(--text3)',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {item.label}
                      </button>
                    );
                  })}

                  {!isMobile && (
                    <div style={{ marginLeft: 'auto', display: 'flex', gap: 8, padding: '5px 0' }}>
                      <button
                        type="button"
                        onClick={run}
                        disabled={busy || !online}
                        title="Run all visible test cases (Ctrl+Enter)"
                        className="ac-hover-surface2"
                        style={{
                          height: 32,
                          padding: '0 14px',
                          border: '1px solid var(--border2)',
                          borderRadius: 8,
                          background: 'var(--surface)',
                          color: 'var(--text)',
                          fontSize: 12.5,
                          fontWeight: 600,
                          cursor: busy ? 'progress' : 'pointer',
                          display: 'flex',
                          alignItems: 'center',
                          gap: 6,
                          opacity: busy || !online ? 0.55 : 1,
                        }}
                      >
                        {running ? <Spinner size={11} /> : <Icon.Play />}
                        Run
                      </button>
                      <button
                        type="button"
                        onClick={submit}
                        disabled={busy || !online}
                        title="Submit to the judge (Ctrl+Shift+Enter)"
                        className="ac-hover-accent ac-active-press"
                        style={{
                          height: 32,
                          padding: '0 16px',
                          border: 'none',
                          borderRadius: 8,
                          background: 'var(--accent)',
                          color: 'var(--accent-ink)',
                          fontSize: 12.5,
                          fontWeight: 600,
                          cursor: busy ? 'progress' : 'pointer',
                          display: 'flex',
                          alignItems: 'center',
                          gap: 6,
                          opacity: busy || !online ? 0.55 : 1,
                        }}
                      >
                        {submitting && <Spinner size={11} color="currentColor" />}
                        Submit
                      </button>
                      <button
                        type="button"
                        onClick={() => setBottomOpen(!bottomOpen)}
                        aria-label={bottomOpen ? 'Collapse panel' : 'Expand panel'}
                        aria-expanded={bottomOpen}
                        className="ac-hover-surface2"
                        style={{ ...iconButton, flexShrink: 0 }}
                      >
                        <span
                          style={{
                            display: 'flex',
                            transform: bottomOpen ? 'none' : 'rotate(180deg)',
                            transition: 'transform .2s',
                          }}
                        >
                          <Icon.Chevron size={13} />
                        </span>
                      </button>
                    </div>
                  )}
                </div>

                {bottomOpen && (
                  <div style={{ overflowY: 'auto', padding: '12px 16px', flex: 1, minHeight: 0 }}>
                    {bottomTab === 'tests' && (
                      <TestsPanel
                        tests={tests}
                        selected={selectedTest}
                        results={runResultsByID}
                        running={running}
                        onSelect={setSelectedTest}
                        onAdd={addTest}
                        onRemove={removeTest}
                        onUpdate={updateTest}
                      />
                    )}

                    {bottomTab === 'console' && (
                      <>
                        {consoleLines.length > 0 ? (
                          <div
                            style={{
                              fontFamily: 'var(--font-mono)',
                              fontSize: 12,
                              lineHeight: 1.8,
                              color: 'var(--code-fg)',
                            }}
                          >
                            {consoleLines.map((line, index) => (
                              <div key={index} style={{ color: line.color, whiteSpace: 'pre-wrap' }}>
                                {line.text}
                              </div>
                            ))}
                          </div>
                        ) : (
                          <p style={{ margin: '8px 0', fontSize: 12.5, color: 'var(--text3)' }}>
                            Run your code to see program output and judge logs here.
                          </p>
                        )}
                      </>
                    )}

                    {bottomTab === 'result' && (
                      <ResultPanel
                        submitting={submitting}
                        checkingResult={checkingResult}
                        submission={submission}
                        streamState={stream.state}
                        liveUpdateError={liveUpdateError || stream.error?.message || ''}
                        detailFetchError={detailFetchError}
                        runResult={runResult}
                        running={running}
                        diagnostics={codeDiagnostics}
                        onReconnectStream={stream.reconnect}
                        onCheckResult={checkResultOnce}
                        onSelectDiagnostic={(diagnostic) => {
                          if (diagnostic.line > 0) {
                            setJumpLine(diagnostic.line);
                            if (stacked) setMobileView('editor');
                          }
                        }}
                        onOpenSubmissions={() => {
                          setTab('submissions');
                          if (stacked) setMobileView('problem');
                        }}
                      />
                    )}
                  </div>
                )}
              </div>

              {isMobile && (
                <div
                  style={{
                    flexShrink: 0,
                    display: 'flex',
                    gap: 10,
                    padding: '10px 14px',
                    borderTop: '1px solid var(--border)',
                    background: 'var(--surface)',
                  }}
                >
                  <button
                    type="button"
                    onClick={run}
                    disabled={busy || !online}
                    className="ac-hover-surface2"
                    style={{
                      flex: 1,
                      minHeight: 44,
                      border: '1px solid var(--border2)',
                      borderRadius: 9,
                      background: 'var(--surface)',
                      color: 'var(--text)',
                      fontSize: 13.5,
                      fontWeight: 600,
                      cursor: 'pointer',
                      opacity: busy || !online ? 0.55 : 1,
                    }}
                  >
                    Run
                  </button>
                  <button
                    type="button"
                    onClick={submit}
                    disabled={busy || !online}
                    className="ac-hover-accent"
                    style={{
                      flex: 2,
                      minHeight: 44,
                      border: 'none',
                      borderRadius: 9,
                      background: 'var(--accent)',
                      color: 'var(--accent-ink)',
                      fontSize: 13.5,
                      fontWeight: 600,
                      cursor: 'pointer',
                      opacity: busy || !online ? 0.55 : 1,
                    }}
                  >
                    Submit
                  </button>
                </div>
              )}
            </div>
          )}
        </div>
      </main>

      {resetDialog && (
        <div
          style={{
            position: 'fixed',
            inset: 0,
            zIndex: 70,
            background: 'rgba(15,10,35,.45)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: 20,
          }}
          onClick={() => setResetDialog(false)}
        >
          <div
            role="dialog"
            aria-modal="true"
            aria-label="Reset code"
            onClick={(event) => event.stopPropagation()}
            style={{
              background: 'var(--surface)',
              border: '1px solid var(--border)',
              borderRadius: 14,
              boxShadow: 'var(--shadow-lg)',
              padding: 22,
              width: '100%',
              maxWidth: 400,
              animation: 'acPop .18s ease',
            }}
          >
            <h2 style={{ margin: '0 0 6px', fontSize: 15.5, fontWeight: 650 }}>Reset your code?</h2>
            <p style={{ margin: '0 0 18px', fontSize: 13, color: 'var(--text2)' }}>
              This replaces the editor contents with the default {langMeta.label} template. Your
              current changes will be lost.
            </p>
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
              <button
                type="button"
                onClick={() => setResetDialog(false)}
                className="ac-hover-surface2"
                style={{
                  height: 36,
                  padding: '0 14px',
                  border: '1px solid var(--border)',
                  borderRadius: 8,
                  background: 'var(--surface)',
                  color: 'var(--text2)',
                  fontSize: 13,
                  fontWeight: 600,
                  cursor: 'pointer',
                }}
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => {
                  setCode(CODE_TEMPLATES[language]);
                  setResetDialog(false);
                  showToast('Code reset to the template', 'info');
                }}
                className="ac-hover-opacity"
                style={{
                  height: 36,
                  padding: '0 14px',
                  border: 'none',
                  borderRadius: 8,
                  background: 'var(--error)',
                  color: '#fff',
                  fontSize: 13,
                  fontWeight: 600,
                  cursor: 'pointer',
                }}
              >
                Reset code
              </button>
            </div>
          </div>
        </div>
      )}

      {restoreSubmission && (
        <div
          style={{
            position: 'fixed',
            inset: 0,
            zIndex: 70,
            background: 'rgba(15,10,35,.45)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: 20,
          }}
          onClick={() => setRestoreSubmission(null)}
        >
          <div
            role="dialog"
            aria-modal="true"
            aria-label="Replace editor code"
            onClick={(event) => event.stopPropagation()}
            style={{
              background: 'var(--surface)',
              border: '1px solid var(--border)',
              borderRadius: 14,
              boxShadow: 'var(--shadow-lg)',
              padding: 22,
              width: '100%',
              maxWidth: 420,
              animation: 'acPop .18s ease',
            }}
          >
            <h2 style={{ margin: '0 0 6px', fontSize: 15.5, fontWeight: 650 }}>
              Replace current editor code?
            </h2>
            <p style={{ margin: '0 0 10px', fontSize: 13, color: 'var(--text2)' }}>
              Your current draft will be replaced with submission #{restoreSubmission.id}.
            </p>
            <p
              style={{
                margin: '0 0 18px',
                fontFamily: 'var(--font-mono)',
                fontSize: 11.5,
                color: 'var(--text3)',
              }}
            >
              {languageMeta(restoreSubmission.language).label} · {formatDateTime(restoreSubmission.created_at)}
            </p>
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
              <button
                type="button"
                onClick={() => setRestoreSubmission(null)}
                className="ac-hover-surface2"
                style={{
                  height: 36,
                  padding: '0 14px',
                  border: '1px solid var(--border)',
                  borderRadius: 8,
                  background: 'var(--surface)',
                  color: 'var(--text2)',
                  fontSize: 13,
                  fontWeight: 600,
                  cursor: 'pointer',
                }}
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => restoreSubmissionCode(restoreSubmission)}
                className="ac-hover-accent"
                style={{
                  height: 36,
                  padding: '0 14px',
                  border: 'none',
                  borderRadius: 8,
                  background: 'var(--accent)',
                  color: 'var(--accent-ink)',
                  fontSize: 13,
                  fontWeight: 600,
                  cursor: 'pointer',
                }}
              >
                Use this code
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

/* ------------------------------------------------------------ subpanels */

function TestsPanel({
  tests,
  selected,
  results,
  running,
  onSelect,
  onAdd,
  onRemove,
  onUpdate,
}: {
  tests: TestCase[];
  selected: number;
  results: Map<string, RunTestCaseResult>;
  running: boolean;
  onSelect: (index: number) => void;
  onAdd: () => void;
  onRemove: (index: number) => void;
  onUpdate: (index: number, patch: Partial<TestCase>) => void;
}) {
  const current = tests[selected];
  const currentResult = current ? results.get(current.id) : undefined;

  return (
    <>
      <div
        style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 6, marginBottom: 12 }}
      >
        {tests.map((test, index) => {
          const active = index === selected;
          return (
            <span
              key={index}
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                border: `1px solid ${active ? 'var(--accent)' : 'var(--border)'}`,
                background: active ? 'var(--accent-soft)' : 'var(--surface)',
                borderRadius: 8,
                overflow: 'hidden',
              }}
            >
              <button
                type="button"
                onClick={() => onSelect(index)}
                aria-pressed={active}
                style={{
                  height: 30,
                  padding: '0 11px',
                  border: 'none',
                  background: 'none',
                  fontSize: 12,
                  fontWeight: 600,
                  color: active ? 'var(--accent-fg)' : 'var(--text2)',
                  cursor: 'pointer',
                  fontFamily: 'var(--font-mono)',
                }}
              >
                {test.name}
              </button>
              {!test.fromExample && tests.length > 1 && (
                <button
                  type="button"
                  onClick={() => onRemove(index)}
                  aria-label={`Remove ${test.name}`}
                  className="ac-hover-error"
                  style={{
                    height: 30,
                    width: 24,
                    border: 'none',
                    background: 'none',
                    color: 'var(--text3)',
                    cursor: 'pointer',
                    fontSize: 10,
                  }}
                >
                  ✕
                </button>
              )}
            </span>
          );
        })}
        <button
          type="button"
          onClick={onAdd}
          className="ac-hover-dash"
          style={{
            height: 30,
            padding: '0 11px',
            border: '1px dashed var(--border2)',
            borderRadius: 8,
            background: 'none',
            color: 'var(--text2)',
            fontSize: 12,
            fontWeight: 600,
            cursor: 'pointer',
            whiteSpace: 'nowrap',
          }}
        >
          + Add case
        </button>
      </div>

      {current && (
        <>
          <label style={testLabel}>
            Input — {current.name}
            <textarea
              value={current.input}
              onChange={(event) => onUpdate(selected, { input: event.target.value })}
              readOnly={current.fromExample}
              rows={4}
              spellCheck={false}
              aria-label={`Test input for ${current.name}`}
              className="ac-field"
              style={testField}
            />
          </label>
          <label style={{ ...testLabel, marginTop: 10 }}>
            Expected output
            <textarea
              value={current.expected ?? ''}
              onChange={(event) =>
                onUpdate(selected, {
                  expected:
                    event.target.value === '' && current.kind === 'custom' ? null : event.target.value,
                })
              }
              readOnly={current.fromExample}
              rows={2}
              spellCheck={false}
              aria-label={`Expected output for ${current.name}`}
              className="ac-field"
              style={testField}
            />
          </label>
          <RunCaseDetails result={currentResult} running={running} />
        </>
      )}
    </>
  );
}

function RunCaseDetails({
  result,
  running,
}: {
  result?: RunTestCaseResult;
  running: boolean;
}) {
  if (running) {
    return (
      <div role="status" style={{ marginTop: 12, display: 'flex', alignItems: 'center', gap: 8 }}>
        <Spinner size={12} />
        <span style={{ fontSize: 12, color: 'var(--text3)', fontFamily: 'var(--font-mono)' }}>
          Running all visible test cases…
        </span>
      </div>
    );
  }

  if (!result) return null;

  const ok = result.status === 'accepted' || result.status === 'executed';
  return (
    <div
      style={{
        marginTop: 12,
        border: '1px solid var(--border)',
        borderRadius: 8,
        background: 'var(--code-bg)',
        padding: 10,
        display: 'grid',
        gap: 8,
      }}
    >
      <ResultTile value={formatRunStatus(result.status)} label="Status" />
      <ResultTile value={`${result.execution_time_ms} ms`} label="Runtime" />
      <ResultTile value={`${result.memory_used_kb} KB`} label="Memory" />
      <div>
        <div style={testLabel}>Actual Output</div>
        <pre style={runOutputBlock}>{result.stdout || result.stderr || (ok ? '' : 'No output')}</pre>
      </div>
    </div>
  );
}

function ResultPanel({
  submitting,
  checkingResult,
  submission,
  streamState,
  liveUpdateError,
  detailFetchError,
  runResult,
  running,
  diagnostics,
  onReconnectStream,
  onCheckResult,
  onSelectDiagnostic,
  onOpenSubmissions,
}: {
  submitting: boolean;
  checkingResult: boolean;
  submission: Submission | null;
  streamState: SubmissionStreamState;
  liveUpdateError: string;
  detailFetchError: string;
  runResult: RunResponse | null;
  running: boolean;
  diagnostics: CodeDiagnostic[];
  onReconnectStream: () => void;
  onCheckResult: () => void;
  onSelectDiagnostic: (diagnostic: CodeDiagnostic) => void;
  onOpenSubmissions: () => void;
}) {
  if (running) {
    return (
      <div role="status" style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '10px 0' }}>
        <Spinner />
        <span style={{ fontSize: 12.5, color: 'var(--text2)', fontFamily: 'var(--font-mono)' }}>
          Running your code…
        </span>
      </div>
    );
  }

  if (submission && isPendingStatus(submission.status) && (submitting || liveUpdateError)) {
    const queued = submission.status === 'PENDING';
    const reconnecting = streamState === 'reconnecting';
    const connecting =
      streamState === 'requesting_ticket' || streamState === 'connecting' || streamState === 'open';
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        <div role="status" style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '10px 0' }}>
          {!liveUpdateError && <Spinner color={queued ? 'var(--accent)' : 'var(--success)'} />}
          <span style={{ fontSize: 12.5, color: 'var(--text2)', fontFamily: 'var(--font-mono)' }}>
            {liveUpdateError
              ? `Submission #${submission.id} is saved, but live updates are unavailable.`
              : reconnecting
                ? `Submission #${submission.id} reconnecting to live updates…`
                : connecting
                  ? `Submission #${submission.id} connecting to live updates…`
                  : queued
                    ? `Submission #${submission.id} queued — waiting for a judge worker…`
                    : `Submission #${submission.id} is being judged…`}
          </span>
        </div>

        {liveUpdateError && (
          <div
            role="alert"
            style={{
              display: 'flex',
              flexWrap: 'wrap',
              alignItems: 'center',
              gap: 10,
              padding: '10px 12px',
              border: '1px solid var(--warn)',
              borderRadius: 10,
              background: 'var(--warn-bg)',
            }}
          >
            <span style={{ flex: 1, minWidth: 200, fontSize: 12.5 }}>
              {liveUpdateError} You can reconnect live updates or check the result once.
            </span>
            <button type="button" onClick={onReconnectStream} className="ac-hover-surface2" style={smallActionButton}>
              Reconnect
            </button>
            <button
              type="button"
              onClick={onCheckResult}
              disabled={checkingResult}
              className="ac-hover-surface2"
              style={{ ...smallActionButton, opacity: checkingResult ? 0.6 : 1 }}
            >
              {checkingResult ? 'Checking…' : 'Check result'}
            </button>
            <button type="button" onClick={onOpenSubmissions} className="ac-hover-surface2" style={smallActionButton}>
              View submissions
            </button>
          </div>
        )}
      </div>
    );
  }

  if (detailFetchError && submission) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        <div
          role="alert"
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            alignItems: 'center',
            gap: 10,
            padding: '10px 12px',
            border: '1px solid var(--warn)',
            borderRadius: 10,
            background: 'var(--warn-bg)',
          }}
        >
          <span style={{ flex: 1, minWidth: 200, fontSize: 12.5 }}>{detailFetchError}</span>
          <button
            type="button"
            onClick={onCheckResult}
            disabled={checkingResult}
            className="ac-hover-surface2"
            style={{ ...smallActionButton, opacity: checkingResult ? 0.6 : 1 }}
          >
            {checkingResult ? 'Checking…' : 'Retry detail'}
          </button>
        </div>
      </div>
    );
  }

  if (submission && !isPendingStatus(submission.status)) {
    const verdict = verdictMeta(submission.status);
    const diagnosticOutput = submissionDetailOutput(submission);

    return (
      <div style={{ animation: 'acFadeUp .3s ease' }}>
        <div
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            alignItems: 'center',
            gap: 10,
            padding: '12px 14px',
            border: `1px solid ${verdict.color}`,
            borderRadius: 10,
            background: verdict.bg,
            marginBottom: 12,
          }}
        >
          <span
            aria-hidden="true"
            style={{
              width: 24,
              height: 24,
              borderRadius: verdict.shape,
              background: 'var(--surface)',
              color: verdict.color,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 13,
              fontWeight: 800,
              flexShrink: 0,
            }}
          >
            {verdict.icon}
          </span>
          <span style={{ fontSize: 14, fontWeight: 700, color: verdict.color }}>{verdict.label}</span>
          <span
            style={{
              marginLeft: 'auto',
              fontFamily: 'var(--font-mono)',
              fontSize: 11.5,
              color: verdict.color,
              whiteSpace: 'nowrap',
            }}
          >
            #{submission.id}
          </span>
        </div>

        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit,minmax(150px,1fr))',
            gap: 10,
            marginBottom: 12,
          }}
        >
          <ResultTile value={languageMeta(submission.language).label} label="Language" />
          <ResultTile value={`#${submission.problem_id}`} label={submission.problem_title} />
          <ResultTile value={formatRuntimeMs(submission.execution_time_ms)} label="Runtime" />
          <ResultTile value={formatMemoryKb(submission.memory_used_kb)} label="Memory" />
          <ResultTile
            value={formatTestcaseCount(submission.passed_testcases, submission.total_testcases)}
            label="Test cases"
          />
          <ResultTile value={new Date(submission.updated_at).toLocaleTimeString()} label="Judged at" />
        </div>

        {diagnosticOutput && (
          <div style={{ marginBottom: 12 }}>
            <div style={settingsLabel}>{diagnosticOutput.label}</div>
            <pre style={runOutputBlock}>{diagnosticOutput.output}</pre>
          </div>
        )}

        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
          <button
            type="button"
            onClick={onOpenSubmissions}
            className="ac-hover-surface2"
            style={{
              height: 34,
              padding: '0 14px',
              border: '1px solid var(--border2)',
              borderRadius: 8,
              background: 'var(--surface)',
              color: 'var(--text)',
              fontSize: 12.5,
              fontWeight: 600,
              cursor: 'pointer',
            }}
          >
            View all submissions
          </button>
        </div>
      </div>
    );
  }

  if (runResult) {
    if (runResult.status === 'compile_error') {
      return (
        <div style={{ animation: 'acFadeUp .25s ease' }}>
          <div style={{ marginBottom: 10, fontSize: 13, fontWeight: 650, color: 'var(--warn)' }}>
            Compile Error
          </div>
          <pre style={runOutputBlock}>{runResult.compile_output || 'Compilation failed'}</pre>
          <DiagnosticList diagnostics={diagnostics} onSelect={onSelectDiagnostic} />
        </div>
      );
    }

    const passed = runResult.tests.filter((test) => !isRunFailure(test)).length;
    const allPassed = runResult.tests.length > 0 && passed === runResult.tests.length;
    return (
      <div style={{ animation: 'acFadeUp .25s ease' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 10 }}>
          <span
            style={{
              fontSize: 13,
              fontWeight: 650,
              color: allPassed ? 'var(--success)' : 'var(--error)',
            }}
          >
            {allPassed ? '✓ All test cases passed' : `✕ ${runResult.tests.length - passed} failed`}
          </span>
          <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text3)' }}>
            {passed}/{runResult.tests.length}
          </span>
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          {runResult.tests.map((test, index) => (
            <div
              key={test.id}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 10,
                padding: '8px 10px',
                border: '1px solid var(--border)',
                borderRadius: 8,
                background: 'var(--code-bg)',
              }}
            >
              <span
                role="img"
                aria-label={isRunFailure(test) ? 'Failed' : 'Passed'}
                style={{
                  width: 18,
                  height: 18,
                  borderRadius: isRunFailure(test) ? 4 : '50%',
                  background: isRunFailure(test) ? 'var(--error-bg)' : 'var(--success-bg)',
                  color: isRunFailure(test) ? 'var(--error)' : 'var(--success)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 10,
                  fontWeight: 700,
                  flexShrink: 0,
                }}
              >
                {isRunFailure(test) ? '✕' : '✓'}
              </span>
              <span
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 11,
                  color: 'var(--text3)',
                  flexShrink: 0,
                }}
              >
                {test.id}
              </span>
              <span
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 11.5,
                  color: 'var(--syn-str)',
                  flex: 1,
                  minWidth: 0,
                  whiteSpace: 'nowrap',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                }}
              >
                → {formatRunStatus(test.status)}
              </span>
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 10.5, color: 'var(--text3)' }}>
                {test.execution_time_ms} ms
              </span>
            </div>
          ))}
        </div>
        <DiagnosticList diagnostics={diagnostics} onSelect={onSelectDiagnostic} />
      </div>
    );
  }

  return (
    <p style={{ margin: '8px 0', fontSize: 12.5, color: 'var(--text3)' }}>
      Run your code against the test cases, or submit for the full judge.
    </p>
  );
}

function DiagnosticList({
  diagnostics,
  onSelect,
}: {
  diagnostics: CodeDiagnostic[];
  onSelect: (diagnostic: CodeDiagnostic) => void;
}) {
  if (diagnostics.length === 0) return null;

  return (
    <div
      role="list"
      aria-label="Code diagnostics"
      style={{
        marginTop: 12,
        border: '1px solid var(--error)',
        borderRadius: 10,
        background: 'var(--error-bg)',
        overflow: 'hidden',
      }}
    >
      {diagnostics.map((diagnostic, index) => (
        <button
          key={`${diagnostic.kind}-${diagnostic.testcase_id ?? 'compile'}-${diagnostic.line}-${diagnostic.column}-${index}`}
          type="button"
          role="listitem"
          onClick={() => onSelect(diagnostic)}
          className="ac-hover-surface2"
          style={{
            display: 'flex',
            width: '100%',
            gap: 10,
            alignItems: 'flex-start',
            padding: '9px 11px',
            border: 'none',
            borderTop: index === 0 ? 'none' : '1px solid color-mix(in srgb, var(--error) 25%, transparent)',
            background: 'transparent',
            color: 'var(--text)',
            textAlign: 'left',
            cursor: diagnostic.line > 0 ? 'pointer' : 'default',
          }}
        >
          <span
            aria-hidden="true"
            style={{
              width: 18,
              height: 18,
              borderRadius: 5,
              background: 'var(--surface)',
              color: 'var(--error)',
              display: 'inline-flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 11,
              fontWeight: 800,
              flexShrink: 0,
            }}
          >
            !
          </span>
          <span style={{ flex: 1, minWidth: 0 }}>
            <span style={{ display: 'block', fontFamily: 'var(--font-mono)', fontSize: 11.5, color: 'var(--error)' }}>
              {diagnostic.testcase_id ? `${diagnostic.testcase_id} · ` : ''}
              Line {diagnostic.line || '—'}, Column {diagnostic.column || 1}
            </span>
            <span style={{ display: 'block', marginTop: 2, fontSize: 12.5, color: 'var(--text2)' }}>
              {diagnostic.message}
            </span>
          </span>
        </button>
      ))}
    </div>
  );
}

function ResultTile({ value, label }: { value: string; label: string }) {
  return (
    <div style={{ border: '1px solid var(--border)', borderRadius: 8, padding: '10px 12px' }}>
      <span
        style={{
          display: 'block',
          fontFamily: 'var(--font-mono)',
          fontSize: 15,
          fontWeight: 650,
          whiteSpace: 'nowrap',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
        }}
      >
        {value}
      </span>
      <span
        style={{
          fontSize: 11,
          color: 'var(--text3)',
          display: 'block',
          whiteSpace: 'nowrap',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
        }}
      >
        {label}
      </span>
    </div>
  );
}

function isRunFailure(test: RunTestCaseResult): boolean {
  return !['accepted', 'executed'].includes(test.status);
}

function formatRunStatus(status: string): string {
  return status
    .split('_')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

function submissionDetailOutput(
  submission: Pick<Submission | SubmissionDetail, 'status' | 'compile_output' | 'error_message'>,
): { label: string; output: string } | null {
  if (submission.status === 'COMPILATION_ERROR' && submission.compile_output) {
    return { label: 'Compilation output', output: submission.compile_output };
  }
  if (submission.status === 'RUNTIME_ERROR' && submission.error_message) {
    return { label: 'Runtime error', output: submission.error_message };
  }
  if (submission.status === 'SYSTEM_ERROR' && submission.error_message) {
    return { label: 'System error', output: submission.error_message };
  }
  if (
    (submission.status === 'TIME_LIMIT_EXCEEDED' || submission.status === 'MEMORY_LIMIT_EXCEEDED') &&
    submission.error_message
  ) {
    return { label: 'Message', output: submission.error_message };
  }
  if (submission.status === 'OUTPUT_LIMIT_EXCEEDED' && submission.error_message) {
    return { label: 'Message', output: submission.error_message };
  }
  return null;
}

function collectRunDiagnostics(result: RunResponse): CodeDiagnostic[] {
  const seen = new Set<string>();
  const diagnostics: CodeDiagnostic[] = [];
  const add = (diagnostic: CodeDiagnostic) => {
    if (!diagnostic || diagnostic.line <= 0) return;
    const key = [
      diagnostic.testcase_id ?? '',
      diagnostic.kind,
      diagnostic.line,
      diagnostic.column,
      diagnostic.message,
    ].join('|');
    if (seen.has(key)) return;
    seen.add(key);
    diagnostics.push(diagnostic);
  };

  for (const diagnostic of result.diagnostics ?? []) add(diagnostic);
  for (const test of result.tests ?? []) {
    for (const diagnostic of test.diagnostics ?? []) add(diagnostic);
  }
  return diagnostics;
}

function buildRunConsoleLines(result: RunResponse): { text: string; color: string }[] {
  if (result.status === 'compile_error') {
    return [{ text: result.compile_output || 'Compilation failed', color: 'var(--warn)' }];
  }

  const lines: { text: string; color: string }[] = [];
  for (const test of result.tests) {
    if (test.stdout) lines.push({ text: `[${test.id}] stdout\n${test.stdout}`, color: 'var(--code-fg)' });
    if (test.stderr) lines.push({ text: `[${test.id}] stderr\n${test.stderr}`, color: 'var(--error)' });
  }
  return lines;
}

const iconButton: React.CSSProperties = {
  width: 32,
  height: 32,
  borderRadius: 8,
  border: '1px solid var(--border)',
  background: 'var(--surface)',
  color: 'var(--text2)',
  cursor: 'pointer',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
};

const smallActionButton: React.CSSProperties = {
  height: 32,
  padding: '0 13px',
  border: '1px solid var(--border2)',
  borderRadius: 8,
  background: 'var(--surface)',
  color: 'var(--text)',
  fontSize: 12,
  fontWeight: 600,
  cursor: 'pointer',
};

const settingsLabel: React.CSSProperties = {
  display: 'block',
  fontSize: 11,
  fontWeight: 650,
  color: 'var(--text3)',
  textTransform: 'uppercase',
  letterSpacing: '.06em',
  marginBottom: 6,
};

const testLabel: React.CSSProperties = {
  display: 'block',
  fontSize: 11,
  fontWeight: 650,
  color: 'var(--text3)',
  textTransform: 'uppercase',
  letterSpacing: '.06em',
};

const testField: React.CSSProperties = {
  display: 'block',
  width: '100%',
  boxSizing: 'border-box',
  resize: 'vertical',
  border: '1px solid var(--border)',
  borderRadius: 8,
  background: 'var(--code-bg)',
  color: 'var(--code-fg)',
  fontFamily: 'var(--font-mono)',
  fontSize: 12.5,
  padding: '9px 11px',
  lineHeight: 1.6,
  marginTop: 6,
};

const runOutputBlock: React.CSSProperties = {
  margin: '5px 0 0',
  minHeight: 34,
  maxHeight: 120,
  overflow: 'auto',
  whiteSpace: 'pre-wrap',
  fontFamily: 'var(--font-mono)',
  fontSize: 12,
  lineHeight: 1.6,
  color: 'var(--code-fg)',
};
