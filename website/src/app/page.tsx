'use client';

import Link from 'next/link';
import { useEffect, useMemo, useState } from 'react';
import { AppShell } from '@/components/AppShell';
import { useAuth } from '@/components/AuthProvider';
import { Card, CodePath, EmptyState, Logo, SkeletonBar, Wordmark } from '@/components/ui';
import { fetchProblemIndex, useProgress, type ProblemIndex } from '@/lib/progress';
import {
  dayKey,
  difficultyMeta,
  greeting,
  languageLabel,
  ratingTier,
  timeAgo,
  verdictMeta,
} from '@/lib/format';
import { useViewportWidth } from '@/lib/hooks';
import type { Difficulty, Problem, SubmissionListItem } from '@/lib/types';

const DIFFICULTY_ORDER: Difficulty[] = ['easy', 'medium', 'hard'];

export default function HomePage() {
  const { user, loading: authLoading } = useAuth();
  const width = useViewportWidth();
  const narrow = width < 1000;

  const { progress, loading: progressLoading } = useProgress(Boolean(user));
  const [index, setIndex] = useState<ProblemIndex | null>(null);
  const [indexFailed, setIndexFailed] = useState(false);

  useEffect(() => {
    fetchProblemIndex()
      .then(setIndex)
      .catch(() => setIndexFailed(true));
  }, []);

  /* --------------------------------------------------------- derived data */

  const week = useMemo(() => buildWeek(progress.submissions), [progress.submissions]);
  const streak = useMemo(() => computeStreak(progress.submissions), [progress.submissions]);

  const byDifficulty = useMemo(() => {
    if (!index) return null;
    const solved: Record<Difficulty, number> = { easy: 0, medium: 0, hard: 0 };
    for (const id of progress.solvedIds) {
      const problem = index.byId.get(id);
      const key = problem?.difficulty?.toLowerCase() as Difficulty | undefined;
      if (key && key in solved) solved[key] += 1;
    }
    return solved;
  }, [index, progress.solvedIds]);

  const recommended = useMemo(() => {
    if (!index) return [] as Problem[];
    return index.problems
      .filter((problem) => !progress.solvedIds.has(problem.id))
      .sort(
        (a, b) =>
          DIFFICULTY_ORDER.indexOf(a.difficulty) - DIFFICULTY_ORDER.indexOf(b.difficulty) ||
          a.id - b.id,
      )
      .slice(0, 3);
  }, [index, progress.solvedIds]);

  /** The most recent problem that was attempted but never accepted. */
  const resumeProblem = useMemo(() => {
    if (!index) return null;
    const attempt = progress.submissions.find(
      (submission) => !progress.solvedIds.has(submission.problem_id),
    );
    if (attempt) return index.byId.get(attempt.problem_id) ?? null;
    return recommended[0] ?? null;
  }, [index, progress.submissions, progress.solvedIds, recommended]);

  const recentSubs = progress.submissions.slice(0, 4);
  const loading = authLoading || (Boolean(user) && progressLoading) || (!index && !indexFailed);

  /* ------------------------------------------------------- signed-out view */

  if (!authLoading && !user) {
    return (
      <AppShell>
        <section
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            textAlign: 'center',
            padding: '56px 20px 40px',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 18 }}>
            <Logo size={34} />
            <Wordmark fontSize={26} />
          </div>
          <h1 style={{ margin: '0 0 8px', fontSize: 26, fontWeight: 650, letterSpacing: '-0.02em' }}>
            Solve, submit, and watch the judge work
          </h1>
          <p style={{ margin: '0 0 22px', fontSize: 14, color: 'var(--text2)', maxWidth: 520 }}>
            Read every problem without an account. Sign in to run code, submit to the judge, and track
            your progress.
          </p>
          <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', justifyContent: 'center' }}>
            <Link
              href="/problems"
              className="ac-hover-accent"
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                height: 40,
                padding: '0 20px',
                borderRadius: 9,
                background: 'var(--accent)',
                color: 'var(--accent-ink)',
                fontSize: 13.5,
                fontWeight: 600,
                textDecoration: 'none',
              }}
            >
              Browse problems
            </Link>
            <Link
              href="/login"
              className="ac-hover-surface2"
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                height: 40,
                padding: '0 20px',
                borderRadius: 9,
                border: '1px solid var(--border2)',
                background: 'var(--surface)',
                color: 'var(--text)',
                fontSize: 13.5,
                fontWeight: 600,
                textDecoration: 'none',
              }}
            >
              Sign in
            </Link>
          </div>
          <div style={{ marginTop: 34, opacity: 0.85 }}>
            <CodePath total={5} done={3} width={220} height={28} />
          </div>
        </section>
      </AppShell>
    );
  }

  /* -------------------------------------------------------- signed-in view */

  return (
    <AppShell>
      <div
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          alignItems: 'flex-end',
          justifyContent: 'space-between',
          gap: 14,
          marginBottom: 22,
        }}
      >
        <div>
          <h1 style={{ margin: 0, fontSize: 22, fontWeight: 650, letterSpacing: '-0.02em' }}>
            {greeting()}
            {user ? `, ${(user.full_name || user.username).split(' ')[0]}` : ''}
          </h1>
          <p style={{ margin: '4px 0 0', color: 'var(--text2)', fontSize: 13.5 }}>
            {streak > 0
              ? 'Pick up where you left off — your streak is alive.'
              : 'Submit today to start a streak.'}
          </p>
        </div>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            background: 'var(--surface)',
            border: '1px solid var(--border)',
            borderRadius: 99,
            padding: '7px 14px',
            boxShadow: 'var(--shadow)',
          }}
        >
          <CodePath total={5} done={Math.min(5, streak)} width={52} height={14} />
          <span
            style={{ fontSize: 12.5, fontWeight: 600, color: 'var(--accent-fg)', whiteSpace: 'nowrap' }}
          >
            {streak}-day streak
          </span>
          <span style={{ fontSize: 12, color: 'var(--text3)', whiteSpace: 'nowrap' }}>
            · {week.total} submission{week.total === 1 ? '' : 's'} this week
          </span>
        </div>
      </div>

      <div
        style={{
          display: 'grid',
          gridTemplateColumns: narrow ? '1fr' : '2fr 1fr',
          gap: 16,
          alignItems: 'start',
        }}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16, minWidth: 0 }}>
          <Card label="Continue solving" style={{ display: 'flex', flexWrap: 'wrap', gap: 20 }}>
            <div style={{ flex: 1, minWidth: 250 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 10 }}>
                <span style={eyebrow}>Continue solving</span>
              </div>
              {loading ? (
                <>
                  <SkeletonBar width="55%" height={18} style={{ marginBottom: 10 }} />
                  <SkeletonBar width="35%" />
                </>
              ) : resumeProblem ? (
                <>
                  <h2
                    style={{ margin: '0 0 6px', fontSize: 18, fontWeight: 650, letterSpacing: '-0.01em' }}
                  >
                    {resumeProblem.title}
                  </h2>
                  <div
                    style={{
                      display: 'flex',
                      flexWrap: 'wrap',
                      alignItems: 'center',
                      gap: 8,
                      marginBottom: 16,
                    }}
                  >
                    <span
                      style={{
                        fontSize: 11.5,
                        fontWeight: 600,
                        color: difficultyMeta(resumeProblem.difficulty).color,
                        background: difficultyMeta(resumeProblem.difficulty).bg,
                        borderRadius: 6,
                        padding: '2px 8px',
                      }}
                    >
                      {difficultyMeta(resumeProblem.difficulty).label}
                    </span>
                    {(resumeProblem.tags ?? []).slice(0, 3).map((tag) => (
                      <span
                        key={tag.id}
                        style={{
                          fontSize: 11.5,
                          color: 'var(--text2)',
                          background: 'var(--surface2)',
                          borderRadius: 6,
                          padding: '2px 8px',
                        }}
                      >
                        {tag.name}
                      </span>
                    ))}
                  </div>
                  <Link
                    href={`/problems/${resumeProblem.slug}`}
                    className="ac-hover-accent ac-active-press"
                    style={{
                      display: 'inline-flex',
                      alignItems: 'center',
                      height: 38,
                      padding: '0 18px',
                      borderRadius: 9,
                      background: 'var(--accent)',
                      color: 'var(--accent-ink)',
                      fontSize: 13.5,
                      fontWeight: 600,
                      textDecoration: 'none',
                    }}
                  >
                    Start solving
                  </Link>
                </>
              ) : (
                <p style={{ margin: 0, fontSize: 13, color: 'var(--text2)' }}>
                  Nothing left to pick up — every published problem is solved. 🎉
                </p>
              )}
            </div>

            <div
              aria-label="Submissions this week"
              style={{
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'center',
                gap: 9,
                padding: '6px 4px 6px 20px',
                borderLeft: narrow ? 'none' : '1px solid var(--border)',
                minWidth: 185,
              }}
            >
              <span style={{ ...eyebrow, color: 'var(--text3)' }}>Last 7 days</span>
              <div style={{ display: 'flex', alignItems: 'flex-end', gap: 8, height: 76 }}>
                {week.days.map((day) => (
                  <div
                    key={day.label}
                    style={{
                      flex: 1,
                      display: 'flex',
                      flexDirection: 'column',
                      alignItems: 'center',
                      gap: 5,
                    }}
                  >
                    <div
                      role="img"
                      aria-label={`${day.label}: ${day.count} submissions, ${day.accepted} accepted`}
                      title={`${day.count} submission${day.count === 1 ? '' : 's'} · ${day.accepted} accepted`}
                      style={{
                        width: '100%',
                        maxWidth: 26,
                        height: Math.max(4, Math.min(56, day.count * 11)),
                        borderRadius: '5px 5px 2px 2px',
                        background:
                          day.count === 0
                            ? 'var(--surface3)'
                            : day.count >= 5
                              ? 'var(--accent)'
                              : 'var(--accent-soft2)',
                        transition: 'height .4s ease',
                      }}
                    />
                    <span
                      style={{ fontSize: 10, color: 'var(--text3)', fontFamily: 'var(--font-mono)' }}
                    >
                      {day.label}
                    </span>
                  </div>
                ))}
              </div>
              <span style={{ fontSize: 12, color: 'var(--text2)' }}>
                <strong style={{ color: 'var(--text)' }}>{week.accepted}</strong> accepted of{' '}
                {week.total}
              </span>
            </div>
          </Card>

          <Card label="Progress by difficulty" padding="18px 20px">
            <h3 style={{ margin: '0 0 4px', fontSize: 13.5, fontWeight: 650 }}>Solved problems</h3>
            <p style={{ margin: '0 0 14px', fontSize: 12, color: 'var(--text3)' }}>
              {index ? `${progress.solvedIds.size} of ${index.total} total` : 'Loading catalog…'}
            </p>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
              {DIFFICULTY_ORDER.map((difficulty) => {
                const meta = difficultyMeta(difficulty);
                const solved = byDifficulty?.[difficulty] ?? 0;
                const total = index?.totalsByDifficulty[difficulty] ?? 0;
                const pct = total ? Math.round((solved / total) * 100) : 0;
                return (
                  <div key={difficulty}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 5 }}>
                      <span style={{ fontSize: 12, fontWeight: 600, color: meta.color }}>
                        {meta.label}
                      </span>
                      <span
                        style={{
                          fontSize: 12,
                          color: 'var(--text3)',
                          fontFamily: 'var(--font-mono)',
                        }}
                      >
                        {solved}/{total}
                      </span>
                    </div>
                    <div
                      style={{
                        height: 6,
                        borderRadius: 99,
                        background: 'var(--surface3)',
                        overflow: 'hidden',
                      }}
                    >
                      <div
                        style={{
                          height: '100%',
                          width: `${pct}%`,
                          borderRadius: 99,
                          background: meta.color,
                          transition: 'width .5s ease',
                        }}
                      />
                    </div>
                  </div>
                );
              })}
            </div>
          </Card>

          <Card label="Recent submissions" padding={0} style={{ overflow: 'hidden' }}>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                padding: '16px 20px 12px',
              }}
            >
              <h3 style={{ margin: 0, fontSize: 13.5, fontWeight: 650 }}>Recent submissions</h3>
              <Link href="/submissions" style={{ fontSize: 12.5, fontWeight: 600 }}>
                View all
              </Link>
            </div>

            {loading && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 10, padding: '4px 20px 16px' }}>
                <SkeletonBar height={34} radius={8} />
                <SkeletonBar height={34} radius={8} />
                <SkeletonBar height={34} radius={8} />
              </div>
            )}

            {!loading && recentSubs.length === 0 && (
              <EmptyState
                title="No submissions yet"
                description="Solve your first problem to see activity here."
              />
            )}

            {!loading &&
              recentSubs.map((submission) => {
                const verdict = verdictMeta(submission.status);
                return (
                  <Link
                    key={submission.id}
                    href="/submissions"
                    className="ac-hover-surface2"
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 12,
                      padding: 'var(--rowpad) 20px',
                      borderTop: '1px solid var(--border)',
                      color: 'var(--text)',
                      textDecoration: 'none',
                    }}
                  >
                    <span
                      aria-hidden="true"
                      style={{
                        width: 26,
                        height: 26,
                        borderRadius: 7,
                        background: verdict.bg,
                        color: verdict.color,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        flexShrink: 0,
                        fontSize: 12,
                        fontWeight: 700,
                      }}
                    >
                      {verdict.icon}
                    </span>
                    <span style={{ flex: 1, minWidth: 0 }}>
                      <span
                        style={{
                          display: 'block',
                          fontSize: 13,
                          fontWeight: 550,
                          whiteSpace: 'nowrap',
                          overflow: 'hidden',
                          textOverflow: 'ellipsis',
                        }}
                      >
                        {submission.problem_title}
                      </span>
                      <span style={{ fontSize: 11.5, color: 'var(--text3)' }}>
                        {verdict.label} · {languageLabel(submission.language)}
                      </span>
                    </span>
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 11,
                        color: 'var(--text3)',
                        flexShrink: 0,
                      }}
                    >
                      {timeAgo(submission.created_at)}
                    </span>
                  </Link>
                );
              })}
          </Card>
        </div>

        <section
          aria-label="Your status"
          style={{
            display: 'flex',
            flexDirection: 'column',
            minWidth: 0,
            background: 'var(--surface)',
            border: '1px solid var(--border)',
            borderRadius: 14,
            boxShadow: 'var(--shadow)',
            alignSelf: 'start',
          }}
        >
          <div aria-label="Rating" style={{ padding: '18px 20px' }}>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                marginBottom: 2,
              }}
            >
              <h3 style={{ margin: 0, fontSize: 13.5, fontWeight: 650 }}>Rating</h3>
              <span
                style={{
                  fontSize: 11,
                  fontWeight: 650,
                  color: 'var(--accent-fg)',
                  background: 'var(--accent-soft)',
                  borderRadius: 6,
                  padding: '2px 8px',
                }}
              >
                {ratingTier(user?.rating ?? 0)}
              </span>
            </div>
            <div style={{ display: 'flex', alignItems: 'baseline', gap: 8 }}>
              <span
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 30,
                  fontWeight: 600,
                  letterSpacing: '-0.02em',
                }}
              >
                {user?.rating ?? 0}
              </span>
            </div>
            <p style={{ margin: '8px 0 0', fontSize: 11.5, color: 'var(--text3)' }}>
              Rating history is not exposed by the auth service yet.
            </p>
          </div>

          <div
            aria-label="Recommended problems"
            style={{ padding: '16px 20px 8px', borderTop: '1px solid var(--border)' }}
          >
            <h3 style={{ margin: '0 0 8px', fontSize: 13.5, fontWeight: 650 }}>Recommended for you</h3>
            {recommended.length === 0 && (
              <p style={{ margin: '0 0 10px', fontSize: 12.5, color: 'var(--text3)' }}>
                {index ? 'Nothing left — everything is solved.' : 'Loading…'}
              </p>
            )}
            {recommended.map((problem) => {
              const meta = difficultyMeta(problem.difficulty);
              return (
                <Link
                  key={problem.id}
                  href={`/problems/${problem.slug}`}
                  className="ac-hover-accent-fg"
                  style={{
                    display: 'flex',
                    width: '100%',
                    alignItems: 'center',
                    gap: 10,
                    padding: '9px 0',
                    borderTop: '1px solid var(--border)',
                    color: 'var(--text)',
                    textDecoration: 'none',
                  }}
                >
                  <span
                    style={{
                      flex: 1,
                      fontSize: 13,
                      fontWeight: 550,
                      whiteSpace: 'nowrap',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                    }}
                  >
                    {problem.title}
                  </span>
                  <span style={{ fontSize: 11, fontWeight: 600, color: meta.color }}>{meta.label}</span>
                </Link>
              );
            })}
          </div>

          <div
            aria-label="Milestones"
            style={{ padding: '16px 20px 18px', borderTop: '1px solid var(--border)' }}
          >
            <h3 style={{ margin: '0 0 10px', fontSize: 13.5, fontWeight: 650 }}>Milestones</h3>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              {buildMilestones(progress.solvedIds.size, week.total, streak).map((milestone) => (
                <div key={milestone.label} style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                  <svg width="18" height="18" viewBox="0 0 18 18" aria-hidden="true">
                    <circle
                      cx="9"
                      cy="9"
                      r="7.5"
                      fill="none"
                      stroke={milestone.done ? 'var(--success)' : 'var(--border2)'}
                      strokeWidth="1.6"
                    />
                    <path
                      d="M5.6 9.2 8 11.5 12.5 6.8"
                      stroke={milestone.done ? 'var(--success)' : 'transparent'}
                      strokeWidth="1.8"
                      fill="none"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    />
                  </svg>
                  <span
                    style={{
                      flex: 1,
                      fontSize: 12.5,
                      color: milestone.done ? 'var(--text)' : 'var(--text2)',
                    }}
                  >
                    {milestone.label}
                  </span>
                  <span
                    style={{ fontFamily: 'var(--font-mono)', fontSize: 10.5, color: 'var(--text3)' }}
                  >
                    {milestone.meta}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </section>
      </div>
    </AppShell>
  );
}

/* ------------------------------------------------------------- utilities */

interface WeekDay {
  label: string;
  count: number;
  accepted: number;
}

const DAY_LABELS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

function buildWeek(submissions: SubmissionListItem[]): {
  days: WeekDay[];
  total: number;
  accepted: number;
} {
  const buckets = new Map<string, WeekDay>();
  const days: WeekDay[] = [];

  for (let offset = 6; offset >= 0; offset -= 1) {
    const date = new Date();
    date.setDate(date.getDate() - offset);
    const day: WeekDay = { label: DAY_LABELS[date.getDay()], count: 0, accepted: 0 };
    buckets.set(dayKey(date), day);
    days.push(day);
  }

  let total = 0;
  let accepted = 0;
  for (const submission of submissions) {
    const bucket = buckets.get(dayKey(submission.created_at));
    if (!bucket) continue;
    bucket.count += 1;
    total += 1;
    if (submission.status === 'ACCEPTED') {
      bucket.accepted += 1;
      accepted += 1;
    }
  }

  return { days, total, accepted };
}

/** Consecutive days ending today (or yesterday) that have at least one submission. */
function computeStreak(submissions: SubmissionListItem[]): number {
  if (submissions.length === 0) return 0;
  const active = new Set(submissions.map((submission) => dayKey(submission.created_at)));

  const cursor = new Date();
  if (!active.has(dayKey(cursor))) {
    cursor.setDate(cursor.getDate() - 1);
    if (!active.has(dayKey(cursor))) return 0;
  }

  let streak = 0;
  while (active.has(dayKey(cursor))) {
    streak += 1;
    cursor.setDate(cursor.getDate() - 1);
  }
  return streak;
}

function buildMilestones(solved: number, weekTotal: number, streak: number) {
  const nextSolved = [1, 10, 25, 50, 100, 250, 500].find((step) => solved < step) ?? solved;
  const nextStreak = [3, 7, 14, 30, 100].find((step) => streak < step) ?? streak;

  return [
    {
      label: `Solve ${nextSolved} problem${nextSolved === 1 ? '' : 's'}`,
      meta: `${solved}/${nextSolved}`,
      done: solved >= nextSolved,
    },
    {
      label: `Reach a ${nextStreak}-day streak`,
      meta: `${streak}/${nextStreak}`,
      done: streak >= nextStreak,
    },
    {
      label: 'Submit at least 5 times this week',
      meta: `${weekTotal}/5`,
      done: weekTotal >= 5,
    },
  ];
}

const eyebrow: React.CSSProperties = {
  fontSize: 11,
  fontWeight: 650,
  letterSpacing: '.08em',
  textTransform: 'uppercase',
  color: 'var(--accent-fg)',
};
