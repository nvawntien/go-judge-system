'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useEffect, useMemo, useState } from 'react';
import { AppShell } from '@/components/AppShell';
import { useAuth } from '@/components/AuthProvider';
import { Card, EmptyState } from '@/components/ui';
import { API_BASE_URL } from '@/lib/api';
import {
  LANGUAGES,
  avatarUrl,
  dayKey,
  difficultyMeta,
  formatDate,
  initials,
  languageLabel,
  ratingTier,
  timeAgo,
  verdictMeta,
} from '@/lib/format';
import { fetchProblemIndex, useProgress, type ProblemIndex } from '@/lib/progress';
import type { Difficulty, SubmissionListItem } from '@/lib/types';

const DIFFICULTY_ORDER: Difficulty[] = ['easy', 'medium', 'hard'];

export default function ProfilePage() {
  const router = useRouter();
  const { user, loading: authLoading } = useAuth();
  const { progress } = useProgress(Boolean(user));
  const [index, setIndex] = useState<ProblemIndex | null>(null);

  useEffect(() => {
    if (!authLoading && !user) router.replace('/login?next=/profile');
  }, [authLoading, user, router]);

  useEffect(() => {
    fetchProblemIndex()
      .then(setIndex)
      .catch(() => setIndex(null));
  }, []);

  const stats = useMemo(() => {
    const total = progress.submissions.length;
    const accepted = progress.submissions.filter((s) => s.status === 'ACCEPTED').length;
    return {
      solved: progress.solvedIds.size,
      attempted: progress.attemptedIds.size,
      submissions: total,
      acceptance: total ? Math.round((accepted / total) * 100) : 0,
    };
  }, [progress]);

  const byDifficulty = useMemo(() => {
    const solved: Record<Difficulty, number> = { easy: 0, medium: 0, hard: 0 };
    if (!index) return solved;
    for (const id of progress.solvedIds) {
      const key = index.byId.get(id)?.difficulty?.toLowerCase() as Difficulty | undefined;
      if (key && key in solved) solved[key] += 1;
    }
    return solved;
  }, [index, progress.solvedIds]);

  const languageUsage = useMemo(() => {
    const counts = new Map<string, number>();
    for (const submission of progress.submissions) {
      const code = submission.language.toUpperCase();
      counts.set(code, (counts.get(code) ?? 0) + 1);
    }
    const total = progress.submissions.length || 1;
    return [...counts.entries()]
      .map(([code, count]) => ({
        code,
        label: languageLabel(code),
        pct: Math.round((count / total) * 100),
        color: LANGUAGES.find((l) => l.code === code)?.color ?? 'var(--text3)',
      }))
      .sort((a, b) => b.pct - a.pct);
  }, [progress.submissions]);

  const topTags = useMemo(() => {
    if (!index) return [] as { name: string; count: number }[];
    const counts = new Map<string, number>();
    for (const id of progress.solvedIds) {
      for (const tag of index.byId.get(id)?.tags ?? []) {
        counts.set(tag.name, (counts.get(tag.name) ?? 0) + 1);
      }
    }
    return [...counts.entries()]
      .map(([name, count]) => ({ name, count }))
      .sort((a, b) => b.count - a.count)
      .slice(0, 8);
  }, [index, progress.solvedIds]);

  const calendar = useMemo(() => buildCalendar(progress.submissions), [progress.submissions]);

  if (!user) return null;

  const avatar = avatarUrl(user.avatar_url, API_BASE_URL);

  const heroMeta: { text: string; href?: string }[] = [
    { text: `Joined ${formatDate(user.created_at)}` },
    ...(user.country ? [{ text: user.country }] : []),
    ...(user.school ? [{ text: user.school }] : []),
    ...(user.company ? [{ text: user.company }] : []),
    ...(user.github_url ? [{ text: 'GitHub', href: user.github_url }] : []),
    ...(user.website_url ? [{ text: 'Website', href: user.website_url }] : []),
    ...(user.linkedin_url ? [{ text: 'LinkedIn', href: user.linkedin_url }] : []),
  ];

  return (
    <AppShell maxWidth={1080}>
      <Card
        label="Profile"
        padding={22}
        style={{ marginBottom: 16, display: 'flex', flexWrap: 'wrap', gap: 18, alignItems: 'flex-start' }}
      >
        {avatar ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={avatar}
            alt=""
            width={76}
            height={76}
            style={{
              width: 76,
              height: 76,
              borderRadius: '50%',
              objectFit: 'cover',
              border: '2px solid var(--accent-soft2)',
              flexShrink: 0,
            }}
          />
        ) : (
          <span
            aria-hidden="true"
            style={{
              width: 76,
              height: 76,
              borderRadius: '50%',
              background: 'var(--accent-soft)',
              border: '2px solid var(--accent-soft2)',
              color: 'var(--accent-fg)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 24,
              fontWeight: 650,
              flexShrink: 0,
            }}
          >
            {initials(user.full_name || user.username)}
          </span>
        )}

        <div style={{ flex: 1, minWidth: 220 }}>
          <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 10 }}>
            <h1 style={{ margin: 0, fontSize: 20, fontWeight: 650, letterSpacing: '-0.01em' }}>
              {user.full_name || user.username}
            </h1>
            <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12.5, color: 'var(--text3)' }}>
              @{user.username}
            </span>
            <span
              style={{
                fontSize: 11,
                fontWeight: 650,
                color: 'var(--accent-fg)',
                background: 'var(--accent-soft)',
                borderRadius: 6,
                padding: '2px 9px',
              }}
            >
              {ratingTier(user.rating)}
            </span>
            <span style={{ fontSize: 12, color: 'var(--text2)' }}>
              Rating{' '}
              <strong style={{ fontFamily: 'var(--font-mono)', color: 'var(--text)' }}>
                {user.rating}
              </strong>
            </span>
          </div>

          {user.bio && (
            <p style={{ margin: '8px 0 10px', fontSize: 13, color: 'var(--text2)', maxWidth: 560 }}>
              {user.bio}
            </p>
          )}

          <div
            style={{
              display: 'flex',
              flexWrap: 'wrap',
              alignItems: 'center',
              gap: 8,
              fontSize: 12,
              color: 'var(--text3)',
            }}
          >
            {heroMeta.map((item, index_) => (
              <span key={item.text} style={{ display: 'flex', gap: 8 }}>
                {index_ > 0 && <span aria-hidden="true">·</span>}
                {item.href ? (
                  <a href={item.href} target="_blank" rel="noreferrer noopener" style={{ fontSize: 12 }}>
                    {item.text}
                  </a>
                ) : (
                  <span>{item.text}</span>
                )}
              </span>
            ))}
          </div>
        </div>

        <Link
          href="/settings"
          className="ac-hover-surface2"
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            height: 36,
            padding: '0 14px',
            border: '1px solid var(--border)',
            borderRadius: 8,
            background: 'var(--surface)',
            color: 'var(--text2)',
            fontSize: 12.5,
            fontWeight: 600,
            textDecoration: 'none',
            flexShrink: 0,
          }}
        >
          Edit profile
        </Link>
      </Card>

      <section
        aria-label="Statistics"
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          background: 'var(--surface)',
          border: '1px solid var(--border)',
          borderRadius: 12,
          boxShadow: 'var(--shadow)',
          marginBottom: 16,
          overflow: 'hidden',
        }}
      >
        {[
          { value: stats.solved, label: 'Problems solved' },
          { value: stats.attempted, label: 'Problems attempted' },
          { value: stats.submissions, label: `Submissions${progress.truncated ? ' (last 1000)' : ''}` },
          { value: `${stats.acceptance}%`, label: 'Acceptance rate' },
        ].map((stat) => (
          <div
            key={stat.label}
            style={{
              flex: 1,
              minWidth: 150,
              padding: '14px 18px',
              borderLeft: '1px solid var(--border)',
              marginLeft: -1,
            }}
          >
            <span
              style={{
                display: 'block',
                fontFamily: 'var(--font-mono)',
                fontSize: 20,
                fontWeight: 650,
                letterSpacing: '-0.02em',
              }}
            >
              {stat.value}
            </span>
            <span style={{ fontSize: 11.5, color: 'var(--text3)' }}>{stat.label}</span>
          </div>
        ))}
      </section>

      <Card label="Contribution calendar" padding="18px 20px" style={{ marginBottom: 16 }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: 12,
            gap: 12,
            flexWrap: 'wrap',
          }}
        >
          <h2 style={{ margin: 0, fontSize: 13.5, fontWeight: 650 }}>Solving activity</h2>
          <span style={{ fontSize: 11.5, color: 'var(--text3)' }}>
            Last 52 weeks · {stats.submissions} submissions
          </span>
        </div>
        <div style={{ overflowX: 'auto' }}>
          <div
            style={{
              display: 'grid',
              gridTemplateRows: 'repeat(7,10px)',
              gridAutoFlow: 'column',
              gridAutoColumns: 'minmax(9px,1fr)',
              gap: 3,
              minWidth: 620,
            }}
          >
            {calendar.map((cell) => (
              <span
                key={cell.key}
                title={`${cell.key}: ${cell.count} submission${cell.count === 1 ? '' : 's'}`}
                style={{ borderRadius: 2.5, background: LEVEL_COLORS[cell.level] }}
              />
            ))}
          </div>
        </div>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 5,
            marginTop: 10,
            fontSize: 10.5,
            color: 'var(--text3)',
          }}
        >
          Less
          {LEVEL_COLORS.map((color) => (
            <span
              key={color}
              style={{ width: 10, height: 10, borderRadius: 2.5, background: color }}
            />
          ))}
          More
        </div>
      </Card>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit,minmax(290px,1fr))', gap: 16 }}>
        <Card label="Difficulty breakdown" padding="18px 20px">
          <h2 style={{ margin: '0 0 14px', fontSize: 13.5, fontWeight: 650 }}>Solved by difficulty</h2>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {DIFFICULTY_ORDER.map((difficulty) => {
              const meta = difficultyMeta(difficulty);
              const solved = byDifficulty[difficulty];
              const total = index?.totalsByDifficulty[difficulty] ?? 0;
              const pct = total ? Math.round((solved / total) * 100) : 0;
              return (
                <div key={difficulty}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 5 }}>
                    <span style={{ fontSize: 12, fontWeight: 600, color: meta.color }}>{meta.label}</span>
                    <span
                      style={{ fontSize: 12, color: 'var(--text3)', fontFamily: 'var(--font-mono)' }}
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
                      style={{ height: '100%', width: `${pct}%`, borderRadius: 99, background: meta.color }}
                    />
                  </div>
                </div>
              );
            })}
          </div>

          <h2 style={{ margin: '20px 0 10px', fontSize: 13.5, fontWeight: 650 }}>Languages</h2>
          {languageUsage.length === 0 ? (
            <p style={{ margin: 0, fontSize: 12, color: 'var(--text3)' }}>No submissions yet.</p>
          ) : (
            <>
              <div
                style={{
                  display: 'flex',
                  height: 8,
                  borderRadius: 99,
                  overflow: 'hidden',
                  gap: 2,
                  marginBottom: 10,
                }}
              >
                {languageUsage.map((item) => (
                  <span
                    key={item.code}
                    role="img"
                    aria-label={`${item.label}: ${item.pct} percent`}
                    title={`${item.label} — ${item.pct}%`}
                    style={{ width: `${item.pct}%`, background: item.color }}
                  />
                ))}
              </div>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12 }}>
                {languageUsage.map((item) => (
                  <span
                    key={item.code}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 5,
                      fontSize: 11.5,
                      color: 'var(--text2)',
                    }}
                  >
                    <span
                      style={{ width: 8, height: 8, borderRadius: 2, background: item.color }}
                    />
                    {item.label} {item.pct}%
                  </span>
                ))}
              </div>
            </>
          )}

          <h2 style={{ margin: '20px 0 10px', fontSize: 13.5, fontWeight: 650 }}>Most solved topics</h2>
          {topTags.length === 0 ? (
            <p style={{ margin: 0, fontSize: 12, color: 'var(--text3)' }}>
              Solve a tagged problem to populate this.
            </p>
          ) : (
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
              {topTags.map((tag) => (
                <span
                  key={tag.name}
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: 4,
                    whiteSpace: 'nowrap',
                    fontSize: 11.5,
                    color: 'var(--text2)',
                    background: 'var(--surface2)',
                    border: '1px solid var(--border)',
                    borderRadius: 6,
                    padding: '3px 9px',
                  }}
                >
                  {tag.name}{' '}
                  <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--text3)' }}>
                    {tag.count} solved
                  </span>
                </span>
              ))}
            </div>
          )}
        </Card>

        <Card label="Recent submissions" padding={0} style={{ overflow: 'hidden', alignSelf: 'start' }}>
          <h2 style={{ margin: 0, padding: '16px 20px 10px', fontSize: 13.5, fontWeight: 650 }}>
            Recent submissions
          </h2>
          {progress.submissions.length === 0 ? (
            <EmptyState title="No submissions yet" description="Your judge activity will appear here." />
          ) : (
            progress.submissions.slice(0, 8).map((submission) => {
              const verdict = verdictMeta(submission.status);
              return (
                <div
                  key={submission.id}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 10,
                    padding: '9px 20px',
                    borderTop: '1px solid var(--border)',
                  }}
                >
                  <span
                    aria-hidden="true"
                    style={{
                      width: 22,
                      height: 22,
                      borderRadius: 6,
                      background: verdict.bg,
                      color: verdict.color,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: 11,
                      fontWeight: 700,
                      flexShrink: 0,
                    }}
                  >
                    {verdict.icon}
                  </span>
                  <span style={{ flex: 1, minWidth: 0 }}>
                    <span
                      style={{
                        display: 'block',
                        fontSize: 12.5,
                        fontWeight: 550,
                        whiteSpace: 'nowrap',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                      }}
                    >
                      {submission.problem_title}
                    </span>
                    <span style={{ fontSize: 11, color: 'var(--text3)' }}>
                      {verdict.label} · {languageLabel(submission.language)} ·{' '}
                      {timeAgo(submission.created_at)}
                    </span>
                  </span>
                </div>
              );
            })
          )}
        </Card>
      </div>
    </AppShell>
  );
}

/* ------------------------------------------------------------ calendar */

const LEVEL_COLORS = [
  'var(--surface3)',
  'var(--accent-soft2)',
  '#B4A0F0',
  'var(--accent)',
  'var(--accent-strong)',
];

function buildCalendar(submissions: SubmissionListItem[]) {
  const counts = new Map<string, number>();
  for (const submission of submissions) {
    const key = dayKey(submission.created_at);
    counts.set(key, (counts.get(key) ?? 0) + 1);
  }

  const cells: { key: string; count: number; level: number }[] = [];
  const start = new Date();
  start.setDate(start.getDate() - (52 * 7 - 1));
  // Align the grid so each column is a full Sunday→Saturday week.
  start.setDate(start.getDate() - start.getDay());

  for (let i = 0; i < 52 * 7; i += 1) {
    const date = new Date(start);
    date.setDate(start.getDate() + i);
    const key = dayKey(date);
    const count = counts.get(key) ?? 0;
    const level = count === 0 ? 0 : count < 2 ? 1 : count < 4 ? 2 : count < 7 ? 3 : 4;
    cells.push({ key, count, level });
  }

  return cells;
}
