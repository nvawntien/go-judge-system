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
import { LANGUAGES, isPendingStatus, languageMeta, verdictMeta } from '@/lib/format';
import { useDismissable, useOnline, useViewportWidth } from '@/lib/hooks';
import { fetchProgress, invalidateProgress } from '@/lib/progress';
import { CODE_TEMPLATES, draftKey } from '@/lib/templates';
import type { LanguageCode, Problem, RunResponse, Submission } from '@/lib/types';

type BottomTab = 'tests' | 'console' | 'result';
type LoadState = 'loading' | 'ready' | 'notfound' | 'error';

interface TestCase {
  name: string;
  input: string;
  expected: string;
  /** Examples come from the problem and cannot be removed. */
  fromExample: boolean;
}

const POLL_INTERVAL_MS = 1500;
const POLL_TIMEOUT_MS = 120_000;

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
              ? 'Cannot reach the API gateway'
              : `GET /api/v1/problems/${slug} failed`,
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
  const [fullscreen, setFullscreen] = useState(false);

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
    try {
      localStorage.setItem('astra-lang', next);
    } catch {
      /* ignore */
    }
  };

  /* ---------------------------------------------------------------- tests */

  const [tests, setTests] = useState<TestCase[]>([]);
  const [selectedTest, setSelectedTest] = useState(0);

  useEffect(() => {
    if (!problem) return;
    const fromExamples = (problem.examples ?? []).map((example, index) => ({
      name: `Case ${index + 1}`,
      input: example.input,
      expected: example.output,
      fromExample: true,
    }));
    setTests(
      fromExamples.length > 0
        ? fromExamples
        : [{ name: 'Case 1', input: '', expected: '', fromExample: false }],
    );
    setSelectedTest(0);
  }, [problem]);

  const updateTest = (index: number, patch: Partial<TestCase>) => {
    setTests((current) => current.map((test, i) => (i === index ? { ...test, ...patch } : test)));
  };

  const addTest = () => {
    if (tests.length >= 8) {
      showToast('Maximum of 8 test cases', 'error');
      return;
    }
    setTests((current) => [
      ...current,
      { name: `Case ${current.length + 1}`, input: '', expected: '', fromExample: false },
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
  const [submission, setSubmission] = useState<Submission | null>(null);
  const [pollTimedOut, setPollTimedOut] = useState(false);
  const pollRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(
    () => () => {
      if (pollRef.current) clearTimeout(pollRef.current);
    },
    [],
  );

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

    const test = tests[selectedTest];
    setRunning(true);
    setBottomTab('result');
    setRunResult(null);

    try {
      const result = await submissionApi.run({
        problem_id: problem.id,
        language,
        source_code: code,
        stdin: test?.input ?? '',
      });
      setRunResult(result);
      setConsoleLines([
        ...(result.compile_output
          ? [{ text: result.compile_output, color: 'var(--warn)' }]
          : []),
        ...(result.stdout ? [{ text: result.stdout, color: 'var(--code-fg)' }] : []),
        ...(result.stderr ? [{ text: result.stderr, color: 'var(--error)' }] : []),
      ]);
    } catch (err) {
      if (err instanceof ApiError && err.isUnimplemented) {
        showToast('The judge has no custom-run endpoint yet — use Submit', 'error');
        setConsoleLines([
          {
            text: `POST ${process.env.NEXT_PUBLIC_RUN_ENDPOINT ?? '/api/v1/submissions/run'} → ${err.httpStatus}`,
            color: 'var(--text3)',
          },
          {
            text: 'Custom-test execution is not exposed by the gateway yet. Submit runs the full judge.',
            color: 'var(--warn)',
          },
        ]);
        setBottomTab('console');
      } else if (err instanceof NetworkError) {
        showToast('Cannot reach the API gateway', 'error');
      } else if (err instanceof ApiError) {
        showToast(err.message || 'Run failed', 'error');
      }
    } finally {
      setRunning(false);
    }
  };

  const pollSubmission = useCallback(
    (id: number, startedAt: number) => {
      pollRef.current = setTimeout(async () => {
        try {
          const latest = await submissionApi.get(id);
          setSubmission(latest);

          if (isPendingStatus(latest.status)) {
            if (Date.now() - startedAt > POLL_TIMEOUT_MS) {
              setPollTimedOut(true);
              setSubmitting(false);
              return;
            }
            pollSubmission(id, startedAt);
            return;
          }

          setSubmitting(false);
          const verdict = verdictMeta(latest.status);
          showToast(
            latest.status === 'ACCEPTED' ? 'Accepted' : `Verdict: ${verdict.label}`,
            latest.status === 'ACCEPTED' ? 'success' : 'error',
          );
          invalidateProgress();
          void syncProgress(true);
        } catch (err) {
          setSubmitting(false);
          if (err instanceof NetworkError) {
            showToast('Lost connection while polling the judge', 'error');
          } else if (err instanceof ApiError) {
            showToast(err.message || 'Could not read the submission', 'error');
          }
        }
      }, POLL_INTERVAL_MS);
    },
    [showToast, syncProgress],
  );

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
    setPollTimedOut(false);
    setSubmission(null);
    setRunResult(null);
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
        created_at: created.created_at,
        updated_at: created.created_at,
      });
      pollSubmission(created.id, Date.now());
    } catch (err) {
      setSubmitting(false);
      if (err instanceof NetworkError) {
        showToast('Cannot reach the API gateway', 'error');
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
  const busy = running || submitting;
  const verdict = submission && !isPendingStatus(submission.status) ? verdictMeta(submission.status) : null;

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
                onUseExample={(input, expected) => {
                  setTests((current) => {
                    const next = [
                      ...current,
                      {
                        name: `Case ${current.length + 1}`,
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
                  setJumpLine(null);
                }}
                language={language}
                fontSize={fontSize}
                tabSize={tabSize}
                highlightLine={jumpLine}
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
                        title="Run against the selected test case (Ctrl+Enter)"
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
                        submission={submission}
                        pollTimedOut={pollTimedOut}
                        runResult={runResult}
                        running={running}
                        onOpenSubmissions={() => router.push('/submissions')}
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
    </div>
  );
}

/* ------------------------------------------------------------ subpanels */

function TestsPanel({
  tests,
  selected,
  onSelect,
  onAdd,
  onRemove,
  onUpdate,
}: {
  tests: TestCase[];
  selected: number;
  onSelect: (index: number) => void;
  onAdd: () => void;
  onRemove: (index: number) => void;
  onUpdate: (index: number, patch: Partial<TestCase>) => void;
}) {
  const current = tests[selected];

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
              value={current.expected}
              onChange={(event) => onUpdate(selected, { expected: event.target.value })}
              rows={2}
              spellCheck={false}
              aria-label={`Expected output for ${current.name}`}
              className="ac-field"
              style={testField}
            />
          </label>
        </>
      )}
    </>
  );
}

function ResultPanel({
  submitting,
  submission,
  pollTimedOut,
  runResult,
  running,
  onOpenSubmissions,
}: {
  submitting: boolean;
  submission: Submission | null;
  pollTimedOut: boolean;
  runResult: RunResponse | null;
  running: boolean;
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

  if (submitting && submission) {
    const queued = submission.status === 'PENDING';
    return (
      <div role="status" style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '10px 0' }}>
        <Spinner color={queued ? 'var(--accent)' : 'var(--success)'} />
        <span style={{ fontSize: 12.5, color: 'var(--text2)', fontFamily: 'var(--font-mono)' }}>
          {queued
            ? `Submission #${submission.id} queued — waiting for a judge worker…`
            : `Submission #${submission.id} is being judged…`}
        </span>
      </div>
    );
  }

  if (pollTimedOut && submission) {
    return (
      <div
        role="alert"
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          alignItems: 'center',
          gap: 12,
          padding: '12px 14px',
          border: '1px solid var(--warn)',
          borderRadius: 10,
          background: 'var(--warn-bg)',
        }}
      >
        <span
          aria-hidden="true"
          style={{
            width: 22,
            height: 22,
            borderRadius: 6,
            background: 'var(--surface)',
            color: 'var(--warn)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 12,
            fontWeight: 700,
            flexShrink: 0,
          }}
        >
          !
        </span>
        <span style={{ flex: 1, minWidth: 200, fontSize: 12.5 }}>
          Submission #{submission.id} is still queued after two minutes. It is saved and will run when
          a judge worker picks it up.
        </span>
        <button
          type="button"
          onClick={onOpenSubmissions}
          className="ac-hover-surface2"
          style={{
            height: 32,
            padding: '0 13px',
            border: '1px solid var(--border2)',
            borderRadius: 8,
            background: 'var(--surface)',
            color: 'var(--text)',
            fontSize: 12,
            fontWeight: 600,
            cursor: 'pointer',
          }}
        >
          View submissions
        </button>
      </div>
    );
  }

  if (submission && !isPendingStatus(submission.status)) {
    const verdict = verdictMeta(submission.status);
    const compileError = submission.status === 'COMPILATION_ERROR';

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
          <ResultTile value={new Date(submission.updated_at).toLocaleTimeString()} label="Judged at" />
        </div>

        {compileError && (
          <p style={{ margin: '0 0 8px', fontSize: 12.5, color: 'var(--text2)' }}>
            The compiler rejected your source. The submission API does not return the compiler output,
            so check your code locally or view the submission for details.
          </p>
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
    const passed = runResult.tests.filter((test) => test.passed).length;
    const allPassed = passed === runResult.tests.length;
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
              key={index}
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
                aria-label={test.passed ? 'Passed' : 'Failed'}
                style={{
                  width: 18,
                  height: 18,
                  borderRadius: test.passed ? '50%' : 4,
                  background: test.passed ? 'var(--success-bg)' : 'var(--error-bg)',
                  color: test.passed ? 'var(--success)' : 'var(--error)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 10,
                  fontWeight: 700,
                  flexShrink: 0,
                }}
              >
                {test.passed ? '✓' : '✕'}
              </span>
              <span
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 11,
                  color: 'var(--text3)',
                  flexShrink: 0,
                }}
              >
                {test.name}
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
                → {test.output}
              </span>
              {test.time_ms !== undefined && (
                <span
                  style={{ fontFamily: 'var(--font-mono)', fontSize: 10.5, color: 'var(--text3)' }}
                >
                  {test.time_ms} ms
                </span>
              )}
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <p style={{ margin: '8px 0', fontSize: 12.5, color: 'var(--text3)' }}>
      Run your code against the test cases, or submit for the full judge.
    </p>
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
