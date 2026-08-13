'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { AppShell } from '@/components/AppShell';
import { useAuth } from '@/components/AuthProvider';
import {
  OwnProfileActions,
  ProfileActivity,
  ProfileHero,
  ProfileLanguages,
  ProfileSectionError,
  ProfileStatGrid,
  ProfileStatsLoading,
} from '@/components/profile/ProfileViews';
import { Card, EmptyState, ErrorState, SkeletonBar } from '@/components/ui';
import { submissionApi } from '@/lib/api';
import { languageLabel, timeAgo, verdictMeta } from '@/lib/format';
import type { MyProfileStats, SubmissionListItem } from '@/lib/types';

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
    void submissionApi.getMyProfileStats(controller.signal)
      .then((stats) => { if (!controller.signal.aborted) setProfileStats(stats); })
      .catch(() => { if (!controller.signal.aborted) setStatsError(true); })
      .finally(() => { if (!controller.signal.aborted) setStatsLoading(false); });
    return () => controller.abort();
  }, [user?.id, statsReload]);

  useEffect(() => {
    if (!user) return;
    const controller = new AbortController();
    setRecentLoading(true);
    setRecentError(false);
    setRecentSubmissions([]);
    void submissionApi.listMine({ page: 1, limit: RECENT_SUBMISSION_LIMIT }, controller.signal)
      .then((response) => { if (!controller.signal.aborted) setRecentSubmissions(response.items); })
      .catch(() => { if (!controller.signal.aborted) setRecentError(true); })
      .finally(() => { if (!controller.signal.aborted) setRecentLoading(false); });
    return () => controller.abort();
  }, [user?.id, recentReload]);

  if (!user) return null;

  return (
    <AppShell maxWidth={1180}>
      <div className="ac-profile-page">
        <ProfileHero profile={user} eyebrow="My competitive dashboard" actions={<OwnProfileActions username={user.username} />} />

        {statsLoading ? <ProfileStatsLoading /> : statsError || !profileStats ? (
          <ProfileSectionError
            title="Could not load competitive statistics"
            detail="Your identity and recent submissions are still available."
            onRetry={() => setStatsReload((value) => value + 1)}
          />
        ) : <ProfileStatGrid stats={profileStats} />}

        <div className="ac-profile-content-grid">
          <div className="ac-profile-side-stack">
            {statsLoading ? <ActivityLoading /> : statsError || !profileStats ? null : <ProfileActivity stats={profileStats} />}
            {user.bio && (
              <section className="ac-profile-about" aria-labelledby="profile-about">
                <h2 id="profile-about">About</h2>
                <p>{user.bio}</p>
              </section>
            )}
          </div>

          <div className="ac-profile-side-stack">
            {statsLoading ? <LanguageLoading /> : statsError || !profileStats ? null : <ProfileLanguages stats={profileStats} />}
            <RecentSubmissions
              submissions={recentSubmissions}
              loading={recentLoading}
              failed={recentError}
              onRetry={() => setRecentReload((value) => value + 1)}
            />
          </div>
        </div>
      </div>
    </AppShell>
  );
}

function ActivityLoading() {
  return (
    <section className="ac-profile-panel" aria-label="Loading activity" aria-busy="true">
      <SkeletonBar width={76} height={16} />
      <SkeletonBar width="100%" height={88} style={{ marginTop: 18 }} />
    </section>
  );
}

function LanguageLoading() {
  return (
    <section className="ac-profile-panel" aria-label="Loading language usage" aria-busy="true">
      <SkeletonBar width={94} height={16} />
      <SkeletonBar width="100%" height={9} style={{ marginTop: 18 }} />
      <SkeletonBar width="72%" height={9} style={{ marginTop: 11 }} />
    </section>
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
    <Card label="Recent submissions" padding={0} style={{ overflow: 'hidden' }}>
      <div style={{ padding: '16px 18px 10px' }}>
        <span className="ac-profile-eyebrow">My latest work</span>
        <h2 style={{ margin: '4px 0 0', fontSize: 15, fontWeight: 650 }}>Recent submissions</h2>
      </div>
      {loading ? (
        <div aria-busy="true" aria-label="Loading recent submissions" style={{ padding: '2px 18px 14px' }}>
          {Array.from({ length: 4 }, (_, index) => <SkeletonSubmission key={index} />)}
        </div>
      ) : failed ? (
        <ErrorState title="Could not load recent submissions" detail="Retry to fetch your latest judge activity." onRetry={onRetry} />
      ) : submissions.length === 0 ? (
        <EmptyState
          title="No submissions yet"
          description="Your latest judge activity will appear here."
          action={<Link href="/problems" className="ac-profile-action-link">Browse problems</Link>}
        />
      ) : (
        <div>
          {submissions.map((submission) => {
            const verdict = verdictMeta(submission.status);
            return (
              <div key={submission.id} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 18px', borderTop: '1px solid var(--border)' }}>
                <span aria-hidden="true" style={{ width: 22, height: 22, borderRadius: 6, background: verdict.bg, color: verdict.color, display: 'grid', placeItems: 'center', fontSize: 11, fontWeight: 700, flexShrink: 0 }}>{verdict.icon}</span>
                <span style={{ flex: 1, minWidth: 0 }}>
                  <span style={{ display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontSize: 12.5, fontWeight: 600 }}>{submission.problem_title}</span>
                  <span style={{ display: 'block', marginTop: 2, color: 'var(--text3)', fontSize: 11 }}>{verdict.label} · {languageLabel(submission.language)} · {timeAgo(submission.created_at)}</span>
                </span>
              </div>
            );
          })}
        </div>
      )}
    </Card>
  );
}

function SkeletonSubmission() {
  return <div style={{ padding: '10px 0', borderTop: '1px solid var(--border)' }}><SkeletonBar width="70%" height={12} /><SkeletonBar width="48%" height={10} style={{ marginTop: 7 }} /></div>;
}
