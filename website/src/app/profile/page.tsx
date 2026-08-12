'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useEffect, useMemo, useState } from 'react';
import { AppShell } from '@/components/AppShell';
import { useAuth } from '@/components/AuthProvider';
import { Card, EmptyState, ErrorState, SkeletonBar } from '@/components/ui';
import { API_BASE_URL, submissionApi } from '@/lib/api';
import {
  LANGUAGES,
  avatarUrl,
  formatDate,
  initials,
  languageLabel,
  ratingTier,
  timeAgo,
  verdictMeta,
} from '@/lib/format';
import type { MyProfileStats, ProfileStatsActivity, SubmissionListItem } from '@/lib/types';

const RECENT_SUBMISSION_LIMIT = 8;

export default function ProfilePage() {
  const router = useRouter();
  const { user, loading: authLoading } = useAuth();
  const [profileStats, setProfileStats] = useState<MyProfileStats | null>(null);
  const [statsLoading, setStatsLoading] = useState(true);
  const [statsError, setStatsError] = useState(false);
  const [statsReload, setStatsReload] = useState(0);
  const [recentSubmissions, setRecentSubmissions] = useState<SubmissionListItem[]>([]);
  const [recentLoading, setRecentLoading] = useState(true);
  const [recentError, setRecentError] = useState(false);
  const [recentReload, setRecentReload] = useState(0);

  useEffect(() => {
    if (!authLoading && !user) router.replace('/login?next=/profile');
  }, [authLoading, user, router]);

  useEffect(() => {
    if (!user) return;

    const controller = new AbortController();
    setStatsLoading(true);
    setStatsError(false);
    setProfileStats(null);

    void submissionApi
      .getMyProfileStats(controller.signal)
      .then((stats) => {
        if (!controller.signal.aborted) setProfileStats(stats);
      })
      .catch(() => {
        if (!controller.signal.aborted) setStatsError(true);
      })
      .finally(() => {
        if (!controller.signal.aborted) setStatsLoading(false);
      });

    return () => controller.abort();
  }, [user?.id, statsReload]);

  useEffect(() => {
    if (!user) return;

    const controller = new AbortController();
    setRecentLoading(true);
    setRecentError(false);
    setRecentSubmissions([]);

    void submissionApi
      .listMine({ page: 1, limit: RECENT_SUBMISSION_LIMIT }, controller.signal)
      .then((response) => {
        if (!controller.signal.aborted) setRecentSubmissions(response.items);
      })
      .catch(() => {
        if (!controller.signal.aborted) setRecentError(true);
      })
      .finally(() => {
        if (!controller.signal.aborted) setRecentLoading(false);
      });

    return () => controller.abort();
  }, [user?.id, recentReload]);

  const calendar = useMemo(
    () => (profileStats ? buildCalendar(profileStats.activity) : []),
    [profileStats],
  );
  const languageUsage = useMemo(
    () => (profileStats ? buildLanguageUsage(profileStats) : []),
    [profileStats],
  );

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
            {heroMeta.map((item, index) => (
              <span key={item.text} style={{ display: 'flex', gap: 8 }}>
                {index > 0 && <span aria-hidden="true">·</span>}
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

      {statsLoading ? (
        <ProfileStatsLoading />
      ) : statsError || !profileStats ? (
        <Card label="Profile statistics" padding={0} style={{ marginBottom: 16 }}>
          <ErrorState
            title="Could not load profile statistics"
            detail="Your profile details and recent submissions are still available."
            onRetry={() => setStatsReload((value) => value + 1)}
          />
        </Card>
      ) : (
        <>
          <ProfileStatCards stats={profileStats} />
          <ActivityCalendar calendar={calendar} totalSubmissions={profileStats.total_submissions} />
        </>
      )}

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit,minmax(290px,1fr))', gap: 16 }}>
        {statsLoading ? (
          <LanguageLoading />
        ) : statsError || !profileStats ? (
          <Card label="Language usage" padding={0}>
            <ErrorState
              title="Language statistics unavailable"
              detail="Retry the profile statistics request to load this section."
              onRetry={() => setStatsReload((value) => value + 1)}
            />
          </Card>
        ) : (
          <LanguageUsage languages={languageUsage} />
        )}

        <RecentSubmissions
          submissions={recentSubmissions}
          loading={recentLoading}
          failed={recentError}
          onRetry={() => setRecentReload((value) => value + 1)}
        />
      </div>
    </AppShell>
  );
}

function ProfileStatCards({ stats }: { stats: MyProfileStats }) {
  const cards = [
    { value: stats.solved_problems, label: 'Problems solved' },
    { value: stats.attempted_problems, label: 'Problems attempted' },
    { value: stats.total_submissions, label: 'Submissions' },
    { value: formatAcceptanceRate(stats.acceptance_rate), label: 'Acceptance rate' },
  ];

  return (
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
      {cards.map((card) => (
        <div
          key={card.label}
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
            {card.value}
          </span>
          <span style={{ fontSize: 11.5, color: 'var(--text3)' }}>{card.label}</span>
        </div>
      ))}
    </section>
  );
}

function ProfileStatsLoading() {
  return (
    <>
      <section
        aria-label="Loading profile statistics"
        aria-busy="true"
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
        {Array.from({ length: 4 }, (_, index) => (
          <div
            key={index}
            style={{ flex: 1, minWidth: 150, padding: '16px 18px', borderLeft: '1px solid var(--border)', marginLeft: -1 }}
          >
            <SkeletonBar width={62} height={22} />
            <SkeletonBar width={104} height={11} style={{ marginTop: 9 }} />
          </div>
        ))}
      </section>
      <Card label="Loading solving activity" padding="18px 20px" style={{ marginBottom: 16 }}>
        <SkeletonBar width={180} height={14} />
        <SkeletonBar width="100%" height={88} style={{ marginTop: 18 }} />
      </Card>
    </>
  );
}

function ActivityCalendar({
  calendar,
  totalSubmissions,
}: {
  calendar: CalendarCell[];
  totalSubmissions: number;
}) {
  return (
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
          Last 52 weeks · {totalSubmissions} submissions
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
              role="img"
              aria-label={activityLabel(cell.key, cell.count)}
              title={activityLabel(cell.key, cell.count)}
              style={{ borderRadius: 2.5, background: LEVEL_COLORS[cell.level] }}
            />
          ))}
        </div>
      </div>
      <div
        aria-hidden="true"
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
          <span key={color} style={{ width: 10, height: 10, borderRadius: 2.5, background: color }} />
        ))}
        More
      </div>
    </Card>
  );
}

function LanguageUsage({ languages }: { languages: LanguageUsageItem[] }) {
  return (
    <Card label="Language usage" padding="18px 20px">
      <h2 style={{ margin: '0 0 14px', fontSize: 13.5, fontWeight: 650 }}>Languages</h2>
      {languages.length === 0 ? (
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
            {languages.map((item) => (
              <span
                key={item.code}
                role="img"
                aria-label={`${item.label}: ${item.count} submission${item.count === 1 ? '' : 's'}, ${item.pct}%`}
                title={`${item.label} — ${item.pct}%`}
                style={{ width: `${item.pct}%`, background: item.color }}
              />
            ))}
          </div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12 }}>
            {languages.map((item) => (
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
                <span style={{ width: 8, height: 8, borderRadius: 2, background: item.color }} />
                {item.label} {item.pct}%
              </span>
            ))}
          </div>
        </>
      )}
    </Card>
  );
}

function LanguageLoading() {
  return (
    <Card label="Loading language usage" padding="18px 20px" style={{ minHeight: 142 }}>
      <SkeletonBar width={100} height={14} />
      <SkeletonBar width="100%" height={8} radius={99} style={{ marginTop: 20 }} />
      <SkeletonBar width={180} height={12} style={{ marginTop: 16 }} />
    </Card>
  );
}

function RecentSubmissions({
  submissions,
  loading,
  failed,
  onRetry,
}: {
  submissions: SubmissionListItem[];
  loading: boolean;
  failed: boolean;
  onRetry: () => void;
}) {
  return (
    <Card label="Recent submissions" padding={0} style={{ overflow: 'hidden', alignSelf: 'start' }}>
      <h2 style={{ margin: 0, padding: '16px 20px 10px', fontSize: 13.5, fontWeight: 650 }}>
        Recent submissions
      </h2>
      {loading ? (
        <div aria-busy="true" aria-label="Loading recent submissions" style={{ padding: '4px 20px 16px' }}>
          {Array.from({ length: 4 }, (_, index) => (
            <div key={index} style={{ padding: '10px 0', borderTop: '1px solid var(--border)' }}>
              <SkeletonBar width="65%" height={12} />
              <SkeletonBar width="45%" height={10} style={{ marginTop: 7 }} />
            </div>
          ))}
        </div>
      ) : failed ? (
        <ErrorState
          title="Could not load recent submissions"
          detail="Retry to fetch your latest judge activity."
          onRetry={onRetry}
        />
      ) : submissions.length === 0 ? (
        <EmptyState title="No submissions yet" description="Your judge activity will appear here." />
      ) : (
        submissions.map((submission) => {
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
                  {verdict.label} · {languageLabel(submission.language)} · {timeAgo(submission.created_at)}
                </span>
              </span>
            </div>
          );
        })
      )}
    </Card>
  );
}

/* ------------------------------------------------------------ statistics */

interface LanguageUsageItem {
  code: string;
  label: string;
  count: number;
  pct: number;
  color: string;
}

function buildLanguageUsage(stats: MyProfileStats): LanguageUsageItem[] {
  const total = stats.total_submissions || 1;
  return stats.language_distribution.map(({ language, count }) => {
    const code = language.toUpperCase();
    return {
      code,
      label: languageLabel(code),
      count,
      pct: Math.round((count / total) * 100),
      color: LANGUAGES.find((item) => item.code === code)?.color ?? 'var(--text3)',
    };
  });
}

function formatAcceptanceRate(rate: number) {
  return `${new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 }).format(rate)}%`;
}

/* ------------------------------------------------------------ calendar */

const LEVEL_COLORS = [
  'var(--surface3)',
  'var(--accent-soft2)',
  '#B4A0F0',
  'var(--accent)',
  'var(--accent-strong)',
];

interface CalendarCell {
  key: string;
  count: number;
  level: number;
}

function buildCalendar(activity: ProfileStatsActivity[]): CalendarCell[] {
  const counts = new Map<string, number>();
  for (const { date, count } of activity) {
    counts.set(date, (counts.get(date) ?? 0) + count);
  }

  const today = new Date();
  const start = new Date(Date.UTC(today.getUTCFullYear(), today.getUTCMonth(), today.getUTCDate()));
  start.setUTCDate(start.getUTCDate() - (52 * 7 - 1));
  // Align the grid so each column is a complete Sunday→Saturday UTC week.
  start.setUTCDate(start.getUTCDate() - start.getUTCDay());

  const cells: CalendarCell[] = [];
  for (let index = 0; index < 52 * 7; index += 1) {
    const date = new Date(start);
    date.setUTCDate(start.getUTCDate() + index);
    const key = date.toISOString().slice(0, 10);
    const count = counts.get(key) ?? 0;
    const level = count === 0 ? 0 : count < 2 ? 1 : count < 4 ? 2 : count < 7 ? 3 : 4;
    cells.push({ key, count, level });
  }
  return cells;
}

function activityLabel(date: string, count: number) {
  const formattedDate = new Intl.DateTimeFormat(undefined, {
    dateStyle: 'long',
    timeZone: 'UTC',
  }).format(new Date(`${date}T00:00:00Z`));
  return `${formattedDate}: ${count} submission${count === 1 ? '' : 's'}`;
}
