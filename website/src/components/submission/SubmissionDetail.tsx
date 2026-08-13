'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import { ApiError, NetworkError, submissionApi } from '@/lib/api';
import { formatDateTime, verdictMeta } from '@/lib/format';
import type { Submission } from '@/lib/types';
import { ErrorState, SkeletonBar } from '@/components/ui';

export function SubmissionDetail({ id, standalone = false }: { id: number; standalone?: boolean }) {
  const [detail, setDetail] = useState<Submission | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [reload, setReload] = useState(0);

  useEffect(() => {
    const controller = new AbortController();
    setDetail(null);
    setError(null);

    submissionApi
      .get(id, controller.signal)
      .then((submission) => {
        if (!controller.signal.aborted) setDetail(submission);
      })
      .catch((cause) => {
        if (controller.signal.aborted) return;
        setError(submissionDetailError(cause, id));
      });

    return () => controller.abort();
  }, [id, reload]);

  if (error) {
    if (standalone) {
      return (
        <section className="ac-submission-detail-card" aria-label={`Submission ${id} detail`}>
          <ErrorState
            title="Could not load submission"
            detail={error}
            onRetry={() => setReload((value) => value + 1)}
          />
        </section>
      );
    }

    return (
      <div className="ac-submission-detail-inline-error" role="alert">
        {error}
      </div>
    );
  }

  if (!detail) {
    return standalone ? (
      <section
        className="ac-submission-detail-card ac-submission-detail-loading"
        aria-label={`Loading submission ${id}`}
        aria-busy="true"
      >
        <SkeletonBar width={190} height={13} />
        <SkeletonBar height={260} radius={8} style={{ marginTop: 16 }} />
      </section>
    ) : (
      <div className="ac-submission-detail-inline-loading" aria-label={`Loading submission ${id}`} aria-busy="true">
        <SkeletonBar height={60} radius={8} />
      </div>
    );
  }

  const verdict = verdictMeta(detail.status);

  return (
    <section
      className={standalone ? 'ac-submission-detail-card' : 'ac-submission-detail-inline'}
      aria-label={`Submission ${detail.id} detail`}
    >
      <div className="ac-submission-detail-meta">
        <span>
          Verdict <strong style={{ color: verdict.color }}>{verdict.label}</strong>
        </span>
        <span>
          Submitted <strong>{formatDateTime(detail.created_at)}</strong>
        </span>
        <span>
          Judged <strong>{formatDateTime(detail.updated_at)}</strong>
        </span>
        <Link href={`/problems?search=${encodeURIComponent(detail.problem_title)}`}>
          Open problem →
        </Link>
      </div>

      <pre className="ac-submission-detail-source">{detail.source_code}</pre>
    </section>
  );
}

function submissionDetailError(cause: unknown, id: number) {
  if (cause instanceof NetworkError) return 'Cannot reach the API gateway. Check your connection and try again.';
  if (cause instanceof ApiError && cause.httpStatus === 404) {
    return `Submission #${id} was not found or is not available to this account.`;
  }
  if (cause instanceof ApiError) return `GET /api/v1/submissions/${id} — ${cause.httpStatus} ${cause.message}`;
  return `Could not load submission #${id}.`;
}
