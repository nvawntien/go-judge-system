'use client';

import { ComingSoon } from '@/components/ComingSoon';

export default function DiscussPage() {
  return (
    <ComingSoon
      title="Discussions"
      subtitle="Community threads on problems, editorials, and contests"
      heading="Discussions are not wired up"
      body="No discussion service exists behind the gateway yet. When one lands, problem-level threads will also appear in the workspace's Discussion tab."
      cta={{ label: 'Open a problem', href: '/problems' }}
    />
  );
}
