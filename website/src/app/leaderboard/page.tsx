'use client';

import { ComingSoon } from '@/components/ComingSoon';

export default function LeaderboardPage() {
  return (
    <ComingSoon
      title="Leaderboard"
      subtitle="Global rating ranking"
      heading="No ranking endpoint yet"
      body="The auth service stores a rating per user but does not expose a ranked listing, so there is no honest way to build a leaderboard. Your own rating is on your profile."
      cta={{ label: 'View your profile', href: '/profile' }}
    />
  );
}
