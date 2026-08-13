'use client';

import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useEffect } from 'react';
import { AppShell, PageHeading } from '@/components/AppShell';
import { useAuth } from '@/components/AuthProvider';
import { SubmissionDetail } from '@/components/submission/SubmissionDetail';
import { ErrorState } from '@/components/ui';

export default function SubmissionDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const { user, loading: authLoading } = useAuth();
  const rawID = Array.isArray(params.id) ? params.id[0] : params.id;
  const submissionID = parseSubmissionID(rawID);
  const returnPath = submissionID === null ? '/submissions' : `/submissions/${submissionID}`;

  useEffect(() => {
    if (!authLoading && !user) {
      router.replace(`/login?next=${encodeURIComponent(returnPath)}`);
    }
  }, [authLoading, returnPath, router, user]);

  if (authLoading || !user) return null;

  return (
    <AppShell maxWidth={1100}>
      <PageHeading
        title={submissionID === null ? 'Submission detail' : `Submission #${submissionID}`}
        subtitle="Your submitted source and judge result"
        actions={<Link href="/submissions" className="ac-profile-action-link">← All submissions</Link>}
      />

      {submissionID === null ? (
        <section className="ac-submission-detail-card">
          <ErrorState title="Invalid submission" detail="The submission identifier must be a positive integer." />
        </section>
      ) : (
        <SubmissionDetail id={submissionID} standalone />
      )}
    </AppShell>
  );
}

function parseSubmissionID(value: string | undefined) {
  if (!value || !/^\d+$/.test(value)) return null;
  const id = Number(value);
  return Number.isSafeInteger(id) && id > 0 ? id : null;
}
