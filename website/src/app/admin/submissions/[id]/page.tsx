'use client';

import Link from 'next/link';
import { useParams } from 'next/navigation';
import { useCallback, useEffect, useMemo, useRef, useState, type CSSProperties, type ReactNode } from 'react';
import { AdminApiError, AdminDialog, AdminForbiddenState, AdminLoadingState, AdminPageHeader, adminCard } from '@/components/admin/AdminStates';
import { DateText, LanguageText, StatusPill, adminTd, adminTh } from '@/components/admin/AdminData';
import { AdminIcon } from '@/components/admin/AdminIcons';
import { buttonStyles } from '@/components/ui';
import { formatDateTime } from '@/lib/format';
import { ApiError, NetworkError, adminSubmissionApi } from '@/lib/api';
import type { AdminSubmissionDetail, AdminSubmissionTestResult, SubmissionStatus } from '@/lib/types';

type ErrorKind = 'forbidden' | 'not-found' | 'invalid' | 'generic';

interface LoadError {
  kind: ErrorKind;
  message: string;
}

function errorFromUnknown(err: unknown): LoadError {
  if (err instanceof NetworkError) {
    return { kind: 'generic', message: 'Cannot reach the API gateway.' };
  }
  if (err instanceof ApiError) {
    if (err.httpStatus === 403) return { kind: 'forbidden', message: err.message || 'Forbidden.' };
    if (err.httpStatus === 404) return { kind: 'not-found', message: err.message || 'Submission not found.' };
    return { kind: 'generic', message: err.message || `Request failed with ${err.httpStatus}.` };
  }
  return { kind: 'generic', message: 'Request failed.' };
}

function formatNumber(value: number | null | undefined, suffix = '') {
  if (typeof value !== 'number') return '-';
  return `${value.toLocaleString()}${suffix}`;
}

function shortAttempt(value: string) {
  if (!value) return '-';
  return value.length > 18 ? `${value.slice(0, 8)}...${value.slice(-6)}` : value;
}

function totalText(detail: AdminSubmissionDetail) {
  const total = detail.total_test_count;
  return total === null ? 'unknown' : total.toLocaleString();
}

function noResultsMessage(status: SubmissionStatus) {
  if (status === 'COMPILATION_ERROR') return 'Compilation failed before testcases were executed.';
  if (status === 'PENDING' || status === 'JUDGING') return 'Judge results have not been recorded yet.';
  return 'No testcase metadata is stored for this submission.';
}

function isTerminalStatus(status: SubmissionStatus) {
  return (
    status === 'ACCEPTED' ||
    status === 'WRONG_ANSWER' ||
    status === 'COMPILATION_ERROR' ||
    status === 'RUNTIME_ERROR' ||
    status === 'TIME_LIMIT_EXCEEDED' ||
    status === 'MEMORY_LIMIT_EXCEEDED' ||
    status === 'OUTPUT_LIMIT_EXCEEDED' ||
    status === 'SYSTEM_ERROR'
  );
}

function DetailStat({ label, value, children }: { label: string; value?: ReactNode; children?: ReactNode }) {
  return (
    <div style={{ ...adminCard, padding: 14 }}>
      <div style={{ marginBottom: 6, color: 'var(--text3)', fontSize: 11, fontWeight: 760, textTransform: 'uppercase', letterSpacing: 0 }}>
        {label}
      </div>
      <div style={{ minHeight: 20, fontSize: 13.5, fontWeight: 650 }}>{children ?? value}</div>
    </div>
  );
}

function JudgeMetric({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div style={{ minHeight: 52 }}>
      <div style={{ marginBottom: 5, color: 'var(--text3)', fontSize: 11, fontWeight: 760, textTransform: 'uppercase', letterSpacing: 0 }}>
        {label}
      </div>
      <div style={{ fontSize: 15, fontWeight: 720 }}>{value}</div>
    </div>
  );
}

function DetailSection({ title, children, actions }: { title: string; children: ReactNode; actions?: ReactNode }) {
  return (
    <section style={{ ...adminCard, overflow: 'hidden' }}>
      <div
        style={{
          padding: '12px 14px',
          borderBottom: '1px solid var(--border)',
          display: 'flex',
          flexWrap: 'wrap',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: 10,
        }}
      >
        <h2 style={{ margin: 0, fontSize: 14.5, fontWeight: 700 }}>{title}</h2>
        {actions}
      </div>
      <div style={{ padding: 14 }}>{children}</div>
    </section>
  );
}

function ProvenanceSection({ detail }: { detail: AdminSubmissionDetail }) {
  return (
    <DetailSection title="Attempt provenance">
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(178px, 1fr))', gap: 10 }}>
        <JudgeMetric label="Attempt ID" value={<span style={{ fontFamily: 'var(--font-mono)' }}>{shortAttempt(detail.current_attempt_id)}</span>} />
        <JudgeMetric label="Trigger" value={detail.attempt_trigger ?? 'unknown'} />
        <JudgeMetric label="Triggered by" value={<span style={{ fontFamily: 'var(--font-mono)' }}>{detail.attempt_triggered_by_user_id ?? 'unknown'}</span>} />
        <JudgeMetric label="Attempt created" value={detail.attempt_created_at ? formatDateTime(detail.attempt_created_at) : 'unknown'} />
        <JudgeMetric label="Testcase version" value={detail.testcase_version === null ? 'unknown' : `v${detail.testcase_version}`} />
        <JudgeMetric
          label="Dataset checksum"
          value={
            detail.dataset_checksum ? (
              <span title={detail.dataset_checksum} style={{ fontFamily: 'var(--font-mono)' }}>
                {shortAttempt(detail.dataset_checksum)}
              </span>
            ) : (
              'unknown'
            )
          }
        />
      </div>
    </DetailSection>
  );
}

function SubmissionNotFound({ message }: { message: string }) {
  return (
    <div role="alert" style={{ ...adminCard, padding: '46px 20px', textAlign: 'center' }}>
      <h2 style={{ margin: '0 0 6px', fontSize: 17, fontWeight: 700 }}>Submission not found</h2>
      <p style={{ margin: '0 auto 18px', maxWidth: 420, color: 'var(--text2)', fontSize: 13.5 }}>{message}</p>
      <Link href="/admin/submissions" className="ac-hover-surface2" style={buttonStyles.secondary(38)}>
        Back to Submissions
      </Link>
    </div>
  );
}

function SourceViewer({ detail }: { detail: AdminSubmissionDetail }) {
  const [copied, setCopied] = useState(false);
  const timerRef = useRef<number | null>(null);

  useEffect(() => {
    return () => {
      if (timerRef.current !== null) window.clearTimeout(timerRef.current);
    };
  }, []);

  const copySource = async () => {
    try {
      await navigator.clipboard.writeText(detail.source_code);
      setCopied(true);
      if (timerRef.current !== null) window.clearTimeout(timerRef.current);
      timerRef.current = window.setTimeout(() => setCopied(false), 1400);
    } catch {
      setCopied(false);
    }
  };

  return (
    <DetailSection
      title="Source code"
      actions={
        <button type="button" onClick={copySource} className="ac-hover-surface2" style={buttonStyles.secondary(34)} aria-live="polite">
          {copied ? 'Copied' : 'Copy'}
        </button>
      }
    >
      <div style={{ marginBottom: 8, display: 'flex', gap: 8, alignItems: 'center', color: 'var(--text3)', fontSize: 12 }}>
        <LanguageText code={detail.language} />
        <span style={{ fontFamily: 'var(--font-mono)' }}>#{detail.id}</span>
      </div>
      <pre
        style={{
          margin: 0,
          maxHeight: 520,
          overflow: 'auto',
          whiteSpace: 'pre',
          border: '1px solid var(--border)',
          borderRadius: 8,
          background: 'var(--surface2)',
          padding: 12,
          color: 'var(--text)',
          fontFamily: 'var(--font-mono)',
          fontSize: 12.5,
          lineHeight: 1.55,
        }}
      >
        <code>{detail.source_code}</code>
      </pre>
    </DetailSection>
  );
}

function MessageBlock({ title, message }: { title: string; message: string }) {
  return (
    <DetailSection title={title}>
      <pre
        style={{
          margin: 0,
          maxHeight: 260,
          overflow: 'auto',
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word',
          border: '1px solid var(--border)',
          borderRadius: 8,
          background: 'var(--surface2)',
          padding: 12,
          color: 'var(--text)',
          fontFamily: 'var(--font-mono)',
          fontSize: 12.5,
          lineHeight: 1.55,
        }}
      >
        {message}
      </pre>
    </DetailSection>
  );
}

function TestResults({ detail }: { detail: AdminSubmissionDetail }) {
  if (detail.test_results.length === 0) {
    return (
      <DetailSection title="Test result metadata">
        <p style={{ margin: 0, color: 'var(--text2)', fontSize: 13.5 }}>{noResultsMessage(detail.status)}</p>
      </DetailSection>
    );
  }

  return (
    <DetailSection title="Test result metadata">
      <div style={{ overflowX: 'auto' }}>
        <table style={{ width: '100%', minWidth: 620, borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              <th style={adminTh}>Case</th>
              <th style={adminTh}>Status</th>
              <th style={adminTh}>Runtime</th>
              <th style={adminTh}>Memory</th>
            </tr>
          </thead>
          <tbody>
            {detail.test_results.map((result: AdminSubmissionTestResult) => (
              <tr key={`${result.index}-${result.status}`}>
                <td style={{ ...adminTd, fontFamily: 'var(--font-mono)' }}>#{result.index}</td>
                <td style={adminTd}>
                  <StatusPill status={result.status} />
                </td>
                <td style={adminTd}>{formatNumber(result.runtime_ms, ' ms')}</td>
                <td style={adminTd}>{formatNumber(result.memory_kb, ' KB')}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </DetailSection>
  );
}

const sectionStack: CSSProperties = {
  display: 'grid',
  gap: 14,
};

export default function AdminSubmissionDetailPage() {
  const params = useParams<{ id: string }>();
  const requestSeq = useRef(0);
  const [detail, setDetail] = useState<AdminSubmissionDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<LoadError | null>(null);
  const [rejudgeDialogOpen, setRejudgeDialogOpen] = useState(false);
  const [rejudgePending, setRejudgePending] = useState(false);
  const [rejudgeError, setRejudgeError] = useState<string | null>(null);

  const submissionId = useMemo(() => {
    const id = Number(params.id);
    return Number.isInteger(id) && id > 0 ? id : null;
  }, [params.id]);

  const load = useCallback(
    (signal?: AbortSignal) => {
      if (submissionId === null) {
        setLoading(false);
        setDetail(null);
        setError({ kind: 'invalid', message: 'Submission ID must be a positive number.' });
        return;
      }

      const generation = requestSeq.current + 1;
      requestSeq.current = generation;
      setLoading(true);
      setError(null);

      adminSubmissionApi
        .get(submissionId, signal)
        .then((res) => {
          if (signal?.aborted || requestSeq.current !== generation) return;
          setDetail(res);
        })
        .catch((err) => {
          if (signal?.aborted || requestSeq.current !== generation) return;
          setDetail(null);
          setError(errorFromUnknown(err));
        })
        .finally(() => {
          if (!signal?.aborted && requestSeq.current === generation) setLoading(false);
        });
    },
    [submissionId],
  );

  useEffect(() => {
    const controller = new AbortController();
    load(controller.signal);
    return () => controller.abort();
  }, [load]);

  const handleRefresh = useCallback(() => {
    load();
  }, [load]);

  const handleRejudge = useCallback(async () => {
    if (submissionId === null || rejudgePending) return;
    setRejudgePending(true);
    setRejudgeError(null);
    try {
      const result = await adminSubmissionApi.rejudge(submissionId);
      setDetail((current) => {
        if (!current || current.id !== result.submission_id) return current;
        return {
          ...current,
          status: result.status,
          current_attempt_id: result.attempt_id,
          attempt_trigger: result.attempt_trigger,
          attempt_triggered_by_user_id: result.attempt_triggered_by_user_id,
          attempt_created_at: result.enqueued_at,
          testcase_version: null,
          dataset_checksum: null,
          passed_test_count: 0,
          executed_test_count: 0,
          total_test_count: null,
          runtime_ms: null,
          memory_kb: null,
          compile_message: null,
          judge_message: null,
          updated_at: result.enqueued_at,
          test_results: [],
        };
      });
      setRejudgeDialogOpen(false);
    } catch (err) {
      setRejudgeError(errorFromUnknown(err).message);
    } finally {
      setRejudgePending(false);
    }
  }, [rejudgePending, submissionId]);

  if (loading) {
    return (
      <>
        <AdminPageHeader title="Submission detail" description="Loading submission metadata and source code." />
        <AdminLoadingState title="Loading submission detail" />
      </>
    );
  }

  if (error?.kind === 'forbidden') return <AdminForbiddenState />;

  if (error?.kind === 'not-found' || error?.kind === 'invalid') {
    return (
      <>
        <AdminPageHeader title="Submission detail" description="The requested submission could not be loaded." />
        <SubmissionNotFound message={error.message} />
      </>
    );
  }

  if (error) {
    return (
      <>
        <AdminPageHeader title="Submission detail" description="The admin submission detail API returned an error." />
        <AdminApiError title="Could not load submission" error={error.message} onRetry={() => load()} />
      </>
    );
  }

  if (!detail) {
    return (
      <>
        <AdminPageHeader title="Submission detail" />
        <SubmissionNotFound message="Submission detail is empty." />
      </>
    );
  }

  return (
    <>
      <AdminPageHeader
        title={`Submission #${detail.id}`}
        description={
          <span style={{ display: 'inline-flex', flexWrap: 'wrap', alignItems: 'center', gap: 8 }}>
            <span>Admin submission detail</span>
            <StatusPill status={detail.status} />
            <LanguageText code={detail.language} />
          </span>
        }
        actions={
          <>
            <button type="button" onClick={handleRefresh} className="ac-hover-surface2" style={{ ...buttonStyles.secondary(38), gap: 7 }}>
              <AdminIcon.Refresh size={15} />
              Refresh status
            </button>
            <button
              type="button"
              onClick={() => {
                setRejudgeError(null);
                setRejudgeDialogOpen(true);
              }}
              disabled={!isTerminalStatus(detail.status) || rejudgePending}
              className="ac-hover-accent"
              style={{ ...buttonStyles.primary(38), gap: 7, opacity: isTerminalStatus(detail.status) ? 1 : 0.5 }}
              aria-disabled={!isTerminalStatus(detail.status) || rejudgePending}
            >
              <AdminIcon.Rejudge size={15} />
              Rejudge
            </button>
            <Link href="/admin/submissions" className="ac-hover-surface2" style={buttonStyles.secondary(38)}>
              Back to Submissions
            </Link>
          </>
        }
      />

      <div style={sectionStack}>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(178px, 1fr))', gap: 12 }}>
          <DetailStat label="Status">
            <StatusPill status={detail.status} />
          </DetailStat>
          <DetailStat label="Language">
            <LanguageText code={detail.language} />
          </DetailStat>
          <DetailStat label="Problem">
            <Link href={`/admin/problems/${detail.problem_id}`} className="ac-link">
              {detail.problem_title || `Problem #${detail.problem_id}`}
            </Link>
            <div style={{ marginTop: 2, color: 'var(--text3)', fontFamily: 'var(--font-mono)', fontSize: 11 }}>#{detail.problem_id}</div>
          </DetailStat>
          <DetailStat label="User">
            <span>{detail.username || '-'}</span>
            <div style={{ marginTop: 2, color: 'var(--text3)', fontFamily: 'var(--font-mono)', fontSize: 11 }}>{detail.user_id}</div>
          </DetailStat>
          <DetailStat label="Attempt" value={<span style={{ fontFamily: 'var(--font-mono)' }}>{shortAttempt(detail.current_attempt_id)}</span>} />
          <DetailStat label="Created" value={<DateText value={detail.created_at} />} />
          <DetailStat label="Updated" value={<span style={{ color: 'var(--text2)', fontSize: 12 }}>{formatDateTime(detail.updated_at)}</span>} />
        </div>

        <DetailSection title="Judge summary">
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: 10 }}>
            <JudgeMetric label="Passed" value={detail.passed_test_count.toLocaleString()} />
            <JudgeMetric label="Executed" value={detail.executed_test_count.toLocaleString()} />
            <JudgeMetric label="Total" value={totalText(detail)} />
            <JudgeMetric label="Runtime" value={formatNumber(detail.runtime_ms, ' ms')} />
            <JudgeMetric label="Memory" value={formatNumber(detail.memory_kb, ' KB')} />
          </div>
        </DetailSection>

        <ProvenanceSection detail={detail} />
        <SourceViewer detail={detail} />
        {detail.compile_message && <MessageBlock title="Compile message" message={detail.compile_message} />}
        {detail.judge_message && <MessageBlock title="Judge message" message={detail.judge_message} />}
        <TestResults detail={detail} />
      </div>

      {rejudgeDialogOpen && (
        <AdminDialog title={`Rejudge submission #${detail.id}?`} onClose={() => (rejudgePending ? undefined : setRejudgeDialogOpen(false))}>
          <p style={{ margin: '0 0 12px', color: 'var(--text2)', fontSize: 13.5, lineHeight: 1.55 }}>
            This creates a new judge attempt for the stored source code and language. The rejudge will use the currently active testcase
            dataset for problem #{detail.problem_id}; historical attempt results remain preserved.
          </p>
          {rejudgeError && (
            <div
              role="alert"
              style={{
                marginBottom: 12,
                border: '1px solid var(--danger)',
                borderRadius: 8,
                background: 'var(--danger-bg)',
                color: 'var(--danger)',
                padding: '9px 10px',
                fontSize: 12.5,
              }}
            >
              {rejudgeError}
            </div>
          )}
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <button
              type="button"
              disabled={rejudgePending}
              onClick={() => setRejudgeDialogOpen(false)}
              className="ac-hover-surface2"
              style={{ ...buttonStyles.secondary(38), opacity: rejudgePending ? 0.55 : 1 }}
            >
              Cancel
            </button>
            <button
              type="button"
              disabled={rejudgePending}
              onClick={handleRejudge}
              className="ac-hover-accent"
              style={{ ...buttonStyles.primary(38), gap: 7, opacity: rejudgePending ? 0.65 : 1 }}
            >
              <AdminIcon.Rejudge size={15} />
              {rejudgePending ? 'Enqueuing...' : 'Create attempt'}
            </button>
          </div>
        </AdminDialog>
      )}
    </>
  );
}
