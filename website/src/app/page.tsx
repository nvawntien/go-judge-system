'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import { AppShell, PageHeading } from '@/components/AppShell';
import { useAuth } from '@/components/AuthProvider';
import { CodePath, EmptyState, ErrorState, Logo, SkeletonBar, Wordmark } from '@/components/ui';
import { submissionApi } from '@/lib/api';
import { greeting, languageLabel, timeAgo, verdictMeta } from '@/lib/format';
import type { MyProfileStats, SubmissionListItem } from '@/lib/types';

const RECENT_LIMIT = 4;

export default function HomePage() {
  const { user, loading: authLoading } = useAuth();
  const [stats, setStats] = useState<MyProfileStats | null>(null);
  const [statsError, setStatsError] = useState(false);
  const [recent, setRecent] = useState<SubmissionListItem[]>([]);
  const [recentLoading, setRecentLoading] = useState(true);
  const [recentError, setRecentError] = useState(false);
  const [reload, setReload] = useState(0);

  useEffect(() => {
    if (!user) {
      setStats(null);
      setRecent([]);
      return;
    }

    const controller = new AbortController();
    setStats(null);
    setStatsError(false);
    setRecentLoading(true);
    setRecentError(false);

    void submissionApi.getMyProfileStats(controller.signal)
      .then((value) => { if (!controller.signal.aborted) setStats(value); })
      .catch(() => { if (!controller.signal.aborted) setStatsError(true); });

    void submissionApi.listMine({ page: 1, limit: RECENT_LIMIT }, controller.signal)
      .then((value) => { if (!controller.signal.aborted) setRecent(value.items); })
      .catch(() => { if (!controller.signal.aborted) setRecentError(true); })
      .finally(() => { if (!controller.signal.aborted) setRecentLoading(false); });

    return () => controller.abort();
  }, [reload, user]);

  if (!authLoading && !user) return <SignedOutHome />;
  if (!user) return <HomeLoading />;

  const firstName = (user.full_name || user.username).split(' ')[0];

  return (
    <AppShell>
      <PageHeading
        title={`${greeting()}, ${firstName}`}
        subtitle="Continue solving, review judge results, or inspect your competitive activity."
        actions={<Link href="/problems" className="ac-button ac-button-primary">Solve a problem</Link>}
      />

      <div className="ac-home-dashboard">
        <section className="ac-panel ac-home-overview" aria-labelledby="home-overview-title">
          <div className="ac-section-heading">
            <h2 id="home-overview-title">Competitive overview</h2>
            <Link href="/profile">Open profile</Link>
          </div>

          {statsError ? (
            <ErrorState title="Could not load your statistics" detail="Recent submissions are still available." onRetry={() => setReload((value) => value + 1)} />
          ) : !stats ? (
            <div className="ac-home-stat-grid" aria-label="Loading competitive overview" aria-busy="true">
              {Array.from({ length: 4 }, (_, index) => <SkeletonBar key={index} height={62} radius={8} />)}
            </div>
          ) : (
            <div className="ac-home-stat-grid">
              <HomeStat label="Solved" value={stats.solved_problems} />
              <HomeStat label="Attempted" value={stats.attempted_problems} />
              <HomeStat label="Submissions" value={stats.total_submissions} />
              <HomeStat label="Acceptance" value={`${formatPercent(stats.acceptance_rate)}%`} />
            </div>
          )}
        </section>

        <section className="ac-panel ac-home-recent" aria-labelledby="home-recent-title">
          <div className="ac-section-heading ac-home-section-heading">
            <h2 id="home-recent-title">Recent submissions</h2>
            <Link href="/submissions">View all</Link>
          </div>

          {recentLoading ? (
            <div className="ac-home-recent-loading" aria-busy="true" aria-label="Loading recent submissions">
              {Array.from({ length: 4 }, (_, index) => <SkeletonBar key={index} height={42} radius={6} />)}
            </div>
          ) : recentError ? (
            <ErrorState title="Could not load recent submissions" onRetry={() => setReload((value) => value + 1)} />
          ) : recent.length === 0 ? (
            <EmptyState title="No submissions yet" description="Recent submissions will appear here." action={<Link href="/problems" className="ac-button ac-button-secondary">Browse problems</Link>} />
          ) : (
            <div className="ac-home-submission-list">
              {recent.map((submission) => {
                const verdict = verdictMeta(submission.status);
                return (
                  <Link key={submission.id} href={`/submissions/${submission.id}`} className="ac-home-submission-row">
                    <span className="ac-home-verdict-icon" aria-hidden="true" style={{ color: verdict.color, background: verdict.bg }}>{verdict.icon}</span>
                    <span className="ac-home-submission-problem">
                      <strong>{submission.problem_title}</strong>
                      <small>{verdict.label} · {languageLabel(submission.language)}</small>
                    </span>
                    <time dateTime={submission.created_at}>{timeAgo(submission.created_at)}</time>
                  </Link>
                );
              })}
            </div>
          )}
        </section>

        <nav className="ac-panel ac-home-shortcuts" aria-label="Dashboard shortcuts">
          <HomeShortcut href="/problems" title="Problem catalog" description="Search and filter published challenges." />
          <HomeShortcut href="/submissions" title="Submission history" description="Review verdicts and submitted source." />
          <HomeShortcut href="/settings" title="Profile settings" description="Update your identity, avatar, and security." secondary />
        </nav>
      </div>
    </AppShell>
  );
}

function SignedOutHome() {
  return (
    <AppShell>
      <section className="ac-home-hero">
        <div className="ac-home-hero-brand"><Logo size={34} /><Wordmark fontSize={26} /></div>
        <h1>Solve problems. Submit code. Understand every verdict.</h1>
        <p>Explore the public problem catalog, then sign in when you are ready to run code, submit solutions, and track your progress.</p>
        <div className="ac-home-hero-actions">
          <Link href="/problems" className="ac-button ac-button-primary">Browse problems</Link>
          <Link href="/login" className="ac-button ac-button-secondary">Sign in</Link>
        </div>
        <div className="ac-home-hero-path"><CodePath total={5} done={3} width={180} height={28} /></div>
      </section>

      <section className="ac-home-capabilities" aria-label="AstraCode capabilities">
        <HomeCapability index="01" title="Readable problem workspace">Technical statements, examples, constraints, and the editor stay focused on solving.</HomeCapability>
        <HomeCapability index="02" title="Clear verdict lifecycle">Queued, judging, and final verdicts remain explicit and never rely on color alone.</HomeCapability>
        <HomeCapability index="03" title="Progress in one place">Review solved problems, submission totals, activity, and language usage from your profile.</HomeCapability>
      </section>
    </AppShell>
  );
}

function HomeLoading() {
  return (
    <AppShell>
      <div className="ac-page-header"><div><SkeletonBar width={220} height={26} /><SkeletonBar width={360} height={12} style={{ marginTop: 8 }} /></div></div>
      <div className="ac-home-dashboard"><SkeletonBar height={152} radius={12} /><SkeletonBar height={260} radius={12} /></div>
    </AppShell>
  );
}

function HomeStat({ label, value }: { label: string; value: number | string }) {
  return <div className="ac-home-stat"><span>{label}</span><strong>{typeof value === 'number' ? value.toLocaleString() : value}</strong></div>;
}

function HomeShortcut({ href, title, description, secondary = false }: { href: string; title: string; description: string; secondary?: boolean }) {
  return <Link href={href} className={`ac-home-shortcut${secondary ? ' ac-home-shortcut-secondary' : ''}`}><span><strong>{title}</strong><small>{description}</small></span><span aria-hidden="true">→</span></Link>;
}

function HomeCapability({ index, title, children }: { index: string; title: string; children: React.ReactNode }) {
  return <article><span>{index}</span><h2>{title}</h2><p>{children}</p></article>;
}

function formatPercent(value: number) {
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 1 }).format(value);
}
