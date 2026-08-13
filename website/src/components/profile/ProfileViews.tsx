'use client';

import Link from 'next/link';
import type { ReactNode } from 'react';
import { Card, ErrorState, SkeletonBar } from '@/components/ui';
import { API_BASE_URL } from '@/lib/api';
import { LANGUAGES, avatarUrl, formatDate, initials, languageLabel } from '@/lib/format';
import type { MyProfileStats, ProfileStatsActivity } from '@/lib/types';

type ProfileIdentity = {
  full_name: string;
  username: string;
  rating: number;
  avatar_url?: string | null;
  bio?: string | null;
  country?: string | null;
  school?: string | null;
  company?: string | null;
  github_url?: string | null;
  website_url?: string | null;
  linkedin_url?: string | null;
  created_at: string;
};

export function ProfileHero({
  profile,
  eyebrow,
  actions,
}: {
  profile: ProfileIdentity;
  eyebrow: string;
  actions?: ReactNode;
}) {
  const displayName = profile.full_name || profile.username;
  const avatar = avatarUrl(profile.avatar_url, API_BASE_URL);
  const details = [profile.country, profile.school, profile.company].filter(Boolean) as string[];
  const links = [
    profile.github_url && { label: 'GitHub', href: profile.github_url },
    profile.website_url && { label: 'Website', href: profile.website_url },
    profile.linkedin_url && { label: 'LinkedIn', href: profile.linkedin_url },
  ].filter(Boolean) as { label: string; href: string }[];

  return (
    <section className="ac-profile-hero" aria-label={`${displayName}'s profile`}>
      {avatar ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img className="ac-profile-avatar" src={avatar} alt={`${displayName}'s avatar`} width={84} height={84} />
      ) : (
        <span className="ac-profile-avatar ac-profile-avatar-fallback" aria-label={`${displayName}'s avatar`} role="img">
          {initials(displayName)}
        </span>
      )}

      <div className="ac-profile-identity">
        <span className="ac-profile-eyebrow">{eyebrow}</span>
        <div className="ac-profile-title-row">
          <h1>{displayName}</h1>
          <span className="ac-profile-handle">@{profile.username}</span>
          {profile.rating > 0 && <span className="ac-profile-rating">Rating {profile.rating.toLocaleString()}</span>}
        </div>
        <div className="ac-profile-meta">
          <span>Joined {formatDate(profile.created_at)}</span>
          {details.map((detail) => <span key={detail}>{detail}</span>)}
          {links.map((link) => (
            <a key={link.label} href={link.href} target="_blank" rel="noopener noreferrer" title={link.href}>
              {link.label}
            </a>
          ))}
        </div>
      </div>
      {actions && <div className="ac-profile-actions">{actions}</div>}
    </section>
  );
}

export function ProfileStatGrid({ stats }: { stats: MyProfileStats }) {
  const cards = [
    { value: stats.solved_problems, label: 'Solved' },
    { value: stats.attempted_problems, label: 'Attempted' },
    { value: stats.total_submissions, label: 'Submissions' },
    { value: formatPercent(stats.acceptance_rate), label: 'Acceptance' },
  ];
  return (
    <section className="ac-profile-stats" aria-labelledby="competitive-summary">
      <div className="ac-profile-section-heading">
        <div>
          <span className="ac-profile-eyebrow">Competitive summary</span>
          <h2 id="competitive-summary">Performance at a glance</h2>
        </div>
      </div>
      <div className="ac-profile-stat-grid">
        {cards.map((card, index) => (
          <div key={card.label} className={index === 0 ? 'ac-profile-stat ac-profile-stat-primary' : 'ac-profile-stat'}>
            <strong>{typeof card.value === 'number' ? card.value.toLocaleString() : card.value}</strong>
            <span>{card.label}</span>
          </div>
        ))}
      </div>
    </section>
  );
}

export function ProfileStatsLoading() {
  return (
    <section className="ac-profile-stats" aria-label="Loading competitive statistics" aria-busy="true">
      <SkeletonBar width={142} height={11} />
      <SkeletonBar width={220} height={20} style={{ marginTop: 9 }} />
      <div className="ac-profile-stat-grid" style={{ marginTop: 18 }}>
        {Array.from({ length: 4 }, (_, index) => (
          <div key={index} className="ac-profile-stat">
            <SkeletonBar width={60} height={26} />
            <SkeletonBar width={72} height={11} style={{ marginTop: 8 }} />
          </div>
        ))}
      </div>
    </section>
  );
}

export function ProfileActivity({
  stats,
  emptyCopy = 'No submissions yet. Solve your first problem to start building activity.',
}: {
  stats: MyProfileStats;
  emptyCopy?: string;
}) {
  const calendar = buildCalendar(stats.activity);
  const monthGroups = buildMonthGroups(calendar);
  const hasActivity = stats.activity.some((item) => item.count > 0);
  return (
    <ProfilePanel title="Activity" description="Last 365 days, grouped by UTC day.">
      {hasActivity ? (
        <>
          <div
            className="ac-profile-heatmap-scroll"
            role="group"
            tabIndex={0}
            aria-label="Submission activity calendar. Scroll horizontally to explore all 52 weeks."
            aria-describedby="profile-activity-legend"
          >
            <div className="ac-profile-timeline">
              {monthGroups.map((month) => (
                <div key={month.key} className="ac-profile-month-group">
                  <time dateTime={`${month.key}-01`} aria-label={month.accessibleLabel} title={month.accessibleLabel}>
                    {month.label}
                  </time>
                  <div className="ac-profile-month-grid" aria-label={`${month.accessibleLabel} activity`}>
                    {month.cells.map((cell, index) => cell ? (
                      <span
                        key={cell.key}
                        role="img"
                        aria-label={activityLabel(cell.key, cell.count)}
                        title={activityLabel(cell.key, cell.count)}
                        style={{ background: LEVEL_COLORS[cell.level] }}
                      />
                    ) : (
                      <span key={`${month.key}-empty-${index}`} className="ac-profile-activity-empty" aria-hidden="true" />
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>
          <div id="profile-activity-legend" className="ac-profile-legend" aria-label="Activity intensity: less to more">
            <span>Less</span>
            {LEVEL_COLORS.map((color) => <i key={color} aria-hidden="true" style={{ background: color }} />)}
            <span>More</span>
          </div>
        </>
      ) : (
        <p className="ac-profile-empty-copy">{emptyCopy}</p>
      )}
    </ProfilePanel>
  );
}

export function ProfileLanguages({ stats }: { stats: MyProfileStats }) {
  const languages = buildLanguageUsage(stats);
  return (
    <ProfilePanel title="Languages" description="Based on all submitted solutions.">
      {languages.length === 0 ? (
        <p className="ac-profile-empty-copy">No language activity yet.</p>
      ) : (
        <div className="ac-profile-language-list">
          {languages.map((item) => (
            <div key={item.code} className="ac-profile-language-row">
              <span className="ac-profile-language-label"><i style={{ background: item.color }} />{item.label}</span>
              <span className="ac-profile-language-value">
                {item.count.toLocaleString()} submission{item.count === 1 ? '' : 's'}
              </span>
            </div>
          ))}
        </div>
      )}
    </ProfilePanel>
  );
}

export function ProfilePanel({ title, description, children }: { title: string; description?: string; children: ReactNode }) {
  return (
    <section className="ac-profile-panel" aria-labelledby={`profile-${title.toLowerCase().replace(/\s+/g, '-')}`}>
      <div className="ac-profile-panel-heading">
        <h2 id={`profile-${title.toLowerCase().replace(/\s+/g, '-')}`}>{title}</h2>
        {description && <p>{description}</p>}
      </div>
      {children}
    </section>
  );
}

export function ProfileSectionError({ title, detail, onRetry }: { title: string; detail: string; onRetry: () => void }) {
  return (
    <Card padding={0} style={{ overflow: 'hidden' }}>
      <ErrorState title={title} detail={detail} onRetry={onRetry} />
    </Card>
  );
}

export function PublicPrivacyNote() {
  return (
    <aside className="ac-profile-privacy-note">
      <strong>Submission history is private.</strong>
      <span>Only aggregate competitive statistics are public.</span>
    </aside>
  );
}

export function OwnProfileActions() {
  return (
    <Link href="/settings" className="ac-profile-action-primary">Edit profile</Link>
  );
}

type LanguageUsageItem = { code: string; label: string; count: number; color: string };

function buildLanguageUsage(stats: MyProfileStats): LanguageUsageItem[] {
  return stats.language_distribution.map(({ language, count }) => {
    const code = language.toUpperCase();
    return {
      code,
      label: languageLabel(code),
      count,
      color: LANGUAGES.find((item) => item.code === code)?.color ?? 'var(--text3)',
    };
  });
}

const LEVEL_COLORS = ['var(--surface3)', 'var(--accent-soft2)', '#B4A0F0', 'var(--accent)', 'var(--accent-strong)'];
const ACTIVITY_DAYS = 365;
const DAYS_PER_WEEK = 7;
const SHORT_MONTH_FORMATTER = new Intl.DateTimeFormat('en-US', { month: 'short', timeZone: 'UTC' });
const ACCESSIBLE_MONTH_FORMATTER = new Intl.DateTimeFormat(undefined, { month: 'long', year: 'numeric', timeZone: 'UTC' });

type CalendarCell = { key: string; count: number; level: number };
type MonthGroup = {
  key: string;
  label: string;
  accessibleLabel: string;
  cells: Array<CalendarCell | null>;
};

function buildCalendar(activity: ProfileStatsActivity[]): CalendarCell[] {
  const counts = new Map(activity.map(({ date, count }) => [date, count]));
  const today = new Date();
  const start = new Date(Date.UTC(today.getUTCFullYear(), today.getUTCMonth(), today.getUTCDate()));
  start.setUTCDate(start.getUTCDate() - (ACTIVITY_DAYS - 1));
  return Array.from({ length: ACTIVITY_DAYS }, (_, index) => {
    const date = new Date(start);
    date.setUTCDate(start.getUTCDate() + index);
    const key = date.toISOString().slice(0, 10);
    const count = counts.get(key) ?? 0;
    const level = count === 0 ? 0 : count < 2 ? 1 : count < 4 ? 2 : count < 7 ? 3 : 4;
    return { key, count, level };
  });
}

function buildMonthGroups(calendar: CalendarCell[]): MonthGroup[] {
  const grouped = new Map<string, CalendarCell[]>();

  for (const cell of calendar) {
    const monthKey = cell.key.slice(0, 7);
    const monthCells = grouped.get(monthKey);
    if (monthCells) monthCells.push(cell);
    else grouped.set(monthKey, [cell]);
  }

  return Array.from(grouped, ([key, monthCells]) => {
    const firstDate = new Date(`${monthCells[0].key}T00:00:00Z`);
    const mondayFirstOffset = (firstDate.getUTCDay() + 6) % DAYS_PER_WEEK;
    const occupiedPositions = mondayFirstOffset + monthCells.length;
    const trailingPositions = (DAYS_PER_WEEK - (occupiedPositions % DAYS_PER_WEEK)) % DAYS_PER_WEEK;

    return {
      key,
      label: SHORT_MONTH_FORMATTER.format(firstDate),
      accessibleLabel: ACCESSIBLE_MONTH_FORMATTER.format(firstDate),
      cells: [
        ...Array.from<null>({ length: mondayFirstOffset }).fill(null),
        ...monthCells,
        ...Array.from<null>({ length: trailingPositions }).fill(null),
      ],
    };
  });
}

function activityLabel(date: string, count: number) {
  const readable = new Intl.DateTimeFormat(undefined, { dateStyle: 'long', timeZone: 'UTC' }).format(new Date(`${date}T00:00:00Z`));
  return `${readable}: ${count} submission${count === 1 ? '' : 's'}`;
}

function formatPercent(value: number) {
  return `${new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 }).format(value)}%`;
}
