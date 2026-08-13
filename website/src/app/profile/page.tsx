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
import { EmptyState, ErrorState, SkeletonBar } from '@/components/ui';
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
    <AppShell maxWidth={1240}>
      <div className="ac-profile-page ac-profile-own">
        <ProfileHero profile={user} eyebrow="My competitive dashboard" actions={<OwnProfileActions />} />

        {statsLoading ? <ProfileStatsLoading /> : statsError || !profileStats ? (
          <ProfileSectionError
            title="Could not load competitive statistics"
            detail="Your identity and recent submissions are still available."
            onRetry={() => setStatsReload((value) => value + 1)}
          />
        ) : <ProfileStatGrid stats={profileStats} />}

        <div className="ac-profile-dashboard-grid">
          <div className="ac-profile-grid-module ac-profile-grid-activity">
            {statsLoading ? <ActivityLoading /> : statsError || !profileStats ? null : <ProfileActivity stats={profileStats} />}
          </div>
          <div className="ac-profile-grid-module ac-profile-grid-languages">
            {statsLoading ? <LanguageLoading /> : statsError || !profileStats ? null : <ProfileLanguages stats={profileStats} />}
          </div>
          <div className="ac-profile-grid-module ac-profile-grid-recent">
            <RecentSubmissions
              submissions={recentSubmissions}
              loading={recentLoading}
              failed={recentError}
              onRetry={() => setRecentReload((value) => value + 1)}
            />
          </div>
          {user.bio && (
            <section className="ac-profile-about ac-profile-grid-about" aria-labelledby="profile-about">
              <h2 id="profile-about">About</h2>
              <p>{user.bio}</p>
            </section>
          )}
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
    <section className="ac-profile-panel ac-profile-recent" aria-labelledby="profile-recent-submissions">
      <div className="ac-profile-panel-heading">
        <div>
          <span className="ac-profile-eyebrow">My latest work</span>
          <h2 id="profile-recent-submissions">Recent submissions</h2>
        </div>
        <p>Latest {RECENT_SUBMISSION_LIMIT} judge runs.</p>
      </div>
      {loading ? (
        <div aria-busy="true" aria-label="Loading recent submissions">
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
        <div className="ac-profile-submission-list">
          <div className="ac-profile-submission-head" aria-hidden="true">
            <span>Problem</span>
            <span className="ac-profile-submission-meta">
              <span>Verdict</span>
              <span>Language</span>
              <span>Submitted</span>
            </span>
          </div>
          {submissions.map((submission) => {
            const verdict = verdictMeta(submission.status);
            return (
              <div key={submission.id} className="ac-profile-submission-row">
                <span className="ac-profile-submission-problem">
                  <span aria-hidden="true" className="ac-profile-submission-icon" style={{ background: verdict.bg, color: verdict.color }}>{verdict.icon}</span>
                  <span>{submission.problem_title}</span>
                </span>
                <span className="ac-profile-submission-meta">
                  <span className="ac-profile-submission-verdict" style={{ color: verdict.color }}>{verdict.label}</span>
                  <span>{languageLabel(submission.language)}</span>
                  <time dateTime={submission.created_at}>{timeAgo(submission.created_at)}</time>
                </span>
              </div>
            );
          })}
        </div>
      )}
    </section>
  );
}

function SkeletonSubmission() {
  return <div className="ac-profile-submission-skeleton"><SkeletonBar width="42%" height={12} /><SkeletonBar width="34%" height={10} /></div>;
}
