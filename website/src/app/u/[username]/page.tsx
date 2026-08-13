'use client';

import { useParams } from 'next/navigation';
import { useEffect, useState } from 'react';
import { AppShell } from '@/components/AppShell';
import {
  ProfileActivity,
  ProfileHero,
  ProfileLanguages,
  ProfileSectionError,
  ProfileCompetitiveOverview,
  ProfileStatsLoading,
  PublicPrivacyNote,
} from '@/components/profile/ProfileViews';
import { useToast } from '@/components/ToastProvider';
import { EmptyState, SkeletonBar } from '@/components/ui';
import { ApiError, submissionApi, userApi } from '@/lib/api';
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
    setState('loading');
    void userApi.profile(username, controller.signal)
      .then((response) => { if (!controller.signal.aborted) { setProfile(response); setState('ready'); } })
      .catch((err) => { if (!controller.signal.aborted) setState(err instanceof ApiError && err.isNotFound ? 'notfound' : 'error'); });
    return () => controller.abort();
  }, [username]);

  useEffect(() => {
    if (!username) return;
    const controller = new AbortController();
    setStats(null);
    setStatsLoading(true);
    setStatsError(false);
    void submissionApi.getPublicProfileStats(username, controller.signal)
      .then((response) => { if (!controller.signal.aborted) setStats(response); })
      .catch(() => { if (!controller.signal.aborted) setStatsError(true); })
      .finally(() => { if (!controller.signal.aborted) setStatsLoading(false); });
    return () => controller.abort();
  }, [username, statsReload]);

  if (state === 'loading') return <PublicProfileLoading />;
  if (state !== 'ready' || !profile) {
    return (
      <AppShell maxWidth={900}>
        <EmptyState
          title={state === 'notfound' ? 'User not found' : "Couldn't load this profile"}
          description={state === 'notfound' ? 'This public profile is unavailable.' : 'Check your connection and try again.'}
        />
      </AppShell>
    );
  }

  const copyLink = async () => {
    try {
      await navigator.clipboard.writeText(`${window.location.origin}/u/${encodeURIComponent(profile.username)}`);
      showToast('Profile link copied', 'success');
    } catch {
      showToast('Clipboard is unavailable', 'error');
    }
  };

  return (
    <AppShell maxWidth={1180}>
      <div className="ac-profile-page">
        <ProfileHero
          profile={profile}
          eyebrow="Public competitive profile"
          actions={<button type="button" onClick={copyLink} className="ac-profile-action-link">Copy profile link</button>}
        />

        {statsLoading ? <ProfileStatsLoading /> : statsError || !stats ? (
          <ProfileSectionError title="Could not load competitive statistics" detail="Profile details are still available." onRetry={() => setStatsReload((value) => value + 1)} />
        ) : <ProfileCompetitiveOverview stats={stats} />}

        {statsLoading ? <PublicInsightsLoading /> : statsError || !stats ? null : (
          <div className="ac-profile-content-grid">
            <ProfileActivity stats={stats} emptyCopy="No public competitive activity yet." />
            <ProfileLanguages stats={stats} />
          </div>
        )}

        {profile.bio && <section className="ac-profile-about" aria-labelledby="public-profile-about"><h2 id="public-profile-about">About</h2><p>{profile.bio}</p></section>}
        <PublicPrivacyNote />
      </div>
    </AppShell>
  );
}

function PublicProfileLoading() {
  return (
    <AppShell maxWidth={1180}>
      <div className="ac-profile-page" aria-busy="true" aria-label="Loading public profile">
        <section className="ac-profile-hero"><SkeletonBar width={84} height={84} radius={42} /><div style={{ flex: 1 }}><SkeletonBar width={108} height={11} /><SkeletonBar width="45%" height={30} style={{ marginTop: 9 }} /><SkeletonBar width="70%" height={12} style={{ marginTop: 14 }} /></div></section>
        <ProfileStatsLoading />
      </div>
    </AppShell>
  );
}

function PublicInsightsLoading() {
  return <div className="ac-profile-content-grid"><section className="ac-profile-panel"><SkeletonBar width={80} height={16} /><SkeletonBar width="100%" height={88} style={{ marginTop: 18 }} /></section><section className="ac-profile-panel"><SkeletonBar width={96} height={16} /><SkeletonBar width="100%" height={9} style={{ marginTop: 18 }} /></section></div>;
}
