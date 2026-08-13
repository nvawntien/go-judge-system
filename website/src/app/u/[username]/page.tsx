'use client';

import { useParams } from 'next/navigation';
import { useEffect, useState } from 'react';
import { AppShell } from '@/components/AppShell';
import { useToast } from '@/components/ToastProvider';
import { Card, EmptyState, ErrorState, SkeletonBar } from '@/components/ui';
import { API_BASE_URL, ApiError, submissionApi, userApi } from '@/lib/api';
import { avatarUrl, formatDate, initials, ratingTier } from '@/lib/format';
import type { MyProfileStats, PublicProfile } from '@/lib/types';

export default function PublicProfilePage() {
  const params = useParams<{ username: string }>();
  const username = params?.username ?? '';
  const { showToast } = useToast();

  const [profile, setProfile] = useState<PublicProfile | null>(null);
  const [state, setState] = useState<'loading' | 'ready' | 'notfound' | 'error'>('loading');
  const [stats, setStats] = useState<MyProfileStats | null>(null);
  const [statsLoading, setStatsLoading] = useState(true);
  const [statsError, setStatsError] = useState(false);
  const [statsReload, setStatsReload] = useState(0);

  useEffect(() => {
    if (!username) return;
    const controller = new AbortController();

    userApi
      .profile(username, controller.signal)
      .then((res) => {
        setProfile(res);
        setState('ready');
      })
      .catch((err) => {
        if (controller.signal.aborted) return;
        setState(err instanceof ApiError && err.isNotFound ? 'notfound' : 'error');
      });

    return () => controller.abort();
  }, [username]);

  useEffect(() => {
    if (!username) return;
    const controller = new AbortController();
    setStats(null);
    setStatsLoading(true);
    setStatsError(false);

    void submissionApi
      .getPublicProfileStats(username, controller.signal)
      .then((response) => {
        if (!controller.signal.aborted) setStats(response);
      })
      .catch(() => {
        if (!controller.signal.aborted) setStatsError(true);
      })
      .finally(() => {
        if (!controller.signal.aborted) setStatsLoading(false);
      });

    return () => controller.abort();
  }, [username, statsReload]);

  if (state === 'loading') {
    return (
      <AppShell maxWidth={880}>
        <Card padding={22}>
          <SkeletonBar width="40%" height={20} style={{ marginBottom: 12 }} />
          <SkeletonBar width="70%" />
        </Card>
      </AppShell>
    );
  }

  if (state !== 'ready' || !profile) {
    return (
      <AppShell maxWidth={880}>
        <Card padding={0}>
          <EmptyState
            title={state === 'notfound' ? 'No such user' : "Couldn't load this profile"}
            description={
              state === 'notfound' ? (
                <span style={{ fontFamily: 'var(--font-mono)' }}>@{username}</span>
              ) : (
                'The auth service did not respond.'
              )
            }
          />
        </Card>
      </AppShell>
    );
  }

  const avatar = avatarUrl(profile.avatar_url, API_BASE_URL);
  const links = [
    profile.github_url && { label: 'GitHub', href: profile.github_url },
    profile.website_url && { label: 'Website', href: profile.website_url },
    profile.linkedin_url && { label: 'LinkedIn', href: profile.linkedin_url },
  ].filter(Boolean) as { label: string; href: string }[];

  return (
    <AppShell maxWidth={880}>
      <Card
        label="Profile"
        padding={22}
        style={{ display: 'flex', flexWrap: 'wrap', gap: 18, alignItems: 'flex-start', marginBottom: 16 }}
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
            {initials(profile.full_name || profile.username)}
          </span>
        )}

        <div style={{ flex: 1, minWidth: 220 }}>
          <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 10 }}>
            <h1 style={{ margin: 0, fontSize: 20, fontWeight: 650 }}>
              {profile.full_name || profile.username}
            </h1>
            <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12.5, color: 'var(--text3)' }}>
              @{profile.username}
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
              {ratingTier(profile.rating)}
            </span>
            <span style={{ fontSize: 12, color: 'var(--text2)' }}>
              Rating{' '}
              <strong style={{ fontFamily: 'var(--font-mono)', color: 'var(--text)' }}>
                {profile.rating}
              </strong>
            </span>
          </div>

          {profile.bio && (
            <p style={{ margin: '8px 0 10px', fontSize: 13, color: 'var(--text2)', maxWidth: 560 }}>
              {profile.bio}
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
            <span>Joined {formatDate(profile.created_at)}</span>
            {profile.country && <span>· {profile.country}</span>}
            {profile.school && <span>· {profile.school}</span>}
            {profile.company && <span>· {profile.company}</span>}
            {links.map((link) => (
              <span key={link.label}>
                ·{' '}
                <a href={link.href} target="_blank" rel="noreferrer noopener" style={{ fontSize: 12 }}>
                  {link.label}
                </a>
              </span>
            ))}
          </div>
        </div>

        <button
          type="button"
          onClick={async () => {
            try {
              await navigator.clipboard.writeText(window.location.href);
              showToast('Profile link copied', 'success');
            } catch {
              showToast('Clipboard is unavailable', 'error');
            }
          }}
          className="ac-hover-surface2"
          style={{
            height: 36,
            padding: '0 14px',
            border: '1px solid var(--border)',
            borderRadius: 8,
            background: 'var(--surface)',
            color: 'var(--text2)',
            fontSize: 12.5,
            fontWeight: 600,
            cursor: 'pointer',
            flexShrink: 0,
          }}
        >
          Copy profile link
        </button>
      </Card>

      {statsLoading ? (
        <PublicProfileStatsLoading />
      ) : statsError || !stats ? (
        <Card label="Competitive statistics" padding={0}>
          <ErrorState
            title="Could not load competitive statistics"
            detail="Profile details are still available."
            onRetry={() => setStatsReload((value) => value + 1)}
          />
        </Card>
      ) : (
        <PublicProfileStatCards stats={stats} />
      )}

      <Card padding={0} style={{ marginTop: 16 }}>
        <EmptyState
          title="Submission history is private"
          description="Only aggregate competitive statistics are public; individual submissions remain private."
          nodes={4}
          done={2}
        />
      </Card>
    </AppShell>
  );
}

function PublicProfileStatCards({ stats }: { stats: MyProfileStats }) {
  const cards = [
    { value: stats.solved_problems, label: 'Problems solved' },
    { value: stats.attempted_problems, label: 'Problems attempted' },
    { value: stats.total_submissions, label: 'Submissions' },
    { value: `${stats.acceptance_rate.toFixed(2)}%`, label: 'Acceptance rate' },
  ];

  return (
    <section
      aria-label="Competitive statistics"
      style={{
        display: 'flex',
        flexWrap: 'wrap',
        background: 'var(--surface)',
        border: '1px solid var(--border)',
        borderRadius: 12,
        boxShadow: 'var(--shadow)',
        overflow: 'hidden',
      }}
    >
      {cards.map((card) => (
        <div
          key={card.label}
          style={{ flex: 1, minWidth: 150, padding: '14px 18px', borderLeft: '1px solid var(--border)', marginLeft: -1 }}
        >
          <span style={{ display: 'block', fontFamily: 'var(--font-mono)', fontSize: 20, fontWeight: 650 }}>
            {card.value}
          </span>
          <span style={{ fontSize: 11.5, color: 'var(--text3)' }}>{card.label}</span>
        </div>
      ))}
    </section>
  );
}

function PublicProfileStatsLoading() {
  return (
    <section
      aria-label="Loading competitive statistics"
      aria-busy="true"
      style={{
        display: 'flex',
        flexWrap: 'wrap',
        background: 'var(--surface)',
        border: '1px solid var(--border)',
        borderRadius: 12,
        boxShadow: 'var(--shadow)',
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
  );
}
