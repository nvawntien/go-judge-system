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
  const hasActivity = stats.activity.some((item) => item.count > 0);
  return (
    <ProfilePanel title="Activity" description="Last 52 weeks, grouped by UTC day.">
      {hasActivity ? (
        <>
          <div className="ac-profile-heatmap-scroll" aria-label="Submission activity heatmap">
            <div className="ac-profile-heatmap">
              {calendar.map((cell) => (
                <span
                  key={cell.key}
                  role="img"
                  aria-label={activityLabel(cell.key, cell.count)}
                  title={activityLabel(cell.key, cell.count)}
                  style={{ background: LEVEL_COLORS[cell.level] }}
                />
              ))}
            </div>
          </div>
          <div className="ac-profile-legend" aria-label="Activity intensity: less to more">
            <span>Less</span>
            {LEVEL_COLORS.map((color) => <i key={color} style={{ background: color }} />)}
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
        <p className="ac-profile-empty-copy">No language usage yet.</p>
      ) : (
        <div className="ac-profile-language-list">
          {languages.map((item) => (
            <div key={item.code} className="ac-profile-language-row">
              <span className="ac-profile-language-label"><i style={{ background: item.color }} />{item.label}</span>
              <span className="ac-profile-language-bar" aria-hidden="true"><i style={{ width: `${item.pct}%`, background: item.color }} /></span>
              <span className="ac-profile-language-value">{item.count.toLocaleString()} · {item.pct}%</span>
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

export function OwnProfileActions({ username }: { username: string }) {
  return (
    <>
      <Link href={`/u/${encodeURIComponent(username)}`} className="ac-profile-action-link">View public profile</Link>
      <Link href="/settings" className="ac-profile-action-primary">Edit profile</Link>
    </>
  );
}

type LanguageUsageItem = { code: string; label: string; count: number; pct: number; color: string };

function buildLanguageUsage(stats: MyProfileStats): LanguageUsageItem[] {
  const total = stats.total_submissions || 1;
  return stats.language_distribution
    .map(({ language, count }) => {
      const code = language.toUpperCase();
      return {
        code,
        label: languageLabel(code),
        count,
        pct: Math.round((count / total) * 100),
        color: LANGUAGES.find((item) => item.code === code)?.color ?? 'var(--text3)',
      };
    })
    .sort((a, b) => b.count - a.count || a.label.localeCompare(b.label));
}

const LEVEL_COLORS = ['var(--surface3)', 'var(--accent-soft2)', '#B4A0F0', 'var(--accent)', 'var(--accent-strong)'];

function buildCalendar(activity: ProfileStatsActivity[]) {
  const counts = new Map(activity.map(({ date, count }) => [date, count]));
  const today = new Date();
  const start = new Date(Date.UTC(today.getUTCFullYear(), today.getUTCMonth(), today.getUTCDate()));
  start.setUTCDate(start.getUTCDate() - (52 * 7 - 1) - start.getUTCDay());
  return Array.from({ length: 52 * 7 }, (_, index) => {
    const date = new Date(start);
    date.setUTCDate(start.getUTCDate() + index);
    const key = date.toISOString().slice(0, 10);
    const count = counts.get(key) ?? 0;
    const level = count === 0 ? 0 : count < 2 ? 1 : count < 4 ? 2 : count < 7 ? 3 : 4;
    return { key, count, level };
  });
}

function activityLabel(date: string, count: number) {
  const readable = new Intl.DateTimeFormat(undefined, { dateStyle: 'long', timeZone: 'UTC' }).format(new Date(`${date}T00:00:00Z`));
  return `${readable}: ${count} submission${count === 1 ? '' : 's'}`;
}

function formatPercent(value: number) {
  return `${new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 }).format(value)}%`;
}
