'use client';

import { ComingSoon } from '@/components/ComingSoon';

export default function ContestsPage() {
  return (
    <ComingSoon
      title="Contests"
      subtitle="Rated rounds and contest history"
      heading="Contests are not available yet"
      body="No contest service is deployed behind the gateway — there is no endpoint to list rounds, register, or compute standings. Rated rounds will appear here once one exists."
    />
  );
}
