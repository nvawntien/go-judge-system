'use client';

import Link from 'next/link';
import { useCallback, useEffect, useState } from 'react';
import { RichText } from '@/components/RichText';
import { EmptyState, Icon, SkeletonBar, Spinner } from '@/components/ui';
import { useToast } from '@/components/ToastProvider';
import { ApiError, NetworkError, submissionApi } from '@/lib/api';
import {
  difficultyMeta,
  formatDateTime,
  formatMemoryKb,
  formatMemoryLimit,
  formatRuntimeMs,
  formatTestcaseCount,
  formatTimeLimit,
  isPendingStatus,
  languageLabel,
  timeAgo,
  verdictMeta,
} from '@/lib/format';
import type { Problem, SubmissionDetail, SubmissionListItem } from '@/lib/types';

export type ProblemTab = 'description' | 'submissions' | 'solutions' | 'discussion';

const TABS: { key: ProblemTab; label: string }[] = [
  { key: 'description', label: 'Description' },
  { key: 'submissions', label: 'Submissions' },
  { key: 'solutions', label: 'Solutions' },
  { key: 'discussion', label: 'Discussion' },
];

export function ProblemPanelSkeleton() {
  return (
    <div
      role="status"
      aria-label="Loading problem"
      style={{ padding: 22, display: 'flex', flexDirection: 'column', gap: 12 }}
    >
      <SkeletonBar width="60%" height={20} />
      <SkeletonBar width="95%" />
      <SkeletonBar width="88%" />
      <SkeletonBar width="72%" />
      <SkeletonBar width="100%" height={90} radius={8} />
    </div>
  );
}

export function ProblemNotFound({ slug }: { slug: string }) {
  return (
    <EmptyState
      title="Problem not found"
      description={
        <span style={{ fontFamily: 'var(--font-mono)' }}>
          {slug} does not exist or is not public.
        </span>
      }
      action={
        <Link
          href="/problems"
          className="ac-hover-accent"
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            height: 36,
            padding: '0 16px',
            borderRadius: 8,
            background: 'var(--accent)',
            color: 'var(--accent-ink)',
            fontSize: 13,
            fontWeight: 600,
            textDecoration: 'none',
          }}
        >
          Browse problems
        </Link>
      }
    />
  );
}

export function ProblemPanel({
  problem,
  tab,
  onTabChange,
  solved,
  attempted,
  signedIn,
  onUseExample,
  onUseSubmissionCode,
  historyRefreshKey,
}: {
  problem: Problem;
  tab: ProblemTab;
  onTabChange: (tab: ProblemTab) => void;
  solved: boolean;
  attempted: boolean;
  signedIn: boolean;
  onUseExample: (input: string, expected: string) => void;
  onUseSubmissionCode: (submission: SubmissionDetail) => void;
  historyRefreshKey: number;
}) {
  const { showToast } = useToast();
  const diff = difficultyMeta(problem.difficulty);

  const status = solved ? 'Solved' : attempted ? 'Attempted' : 'Not attempted';
  const statusColor = solved ? 'var(--success)' : attempted ? 'var(--warn)' : 'var(--border2)';
  const statusIcon = solved ? '✓' : attempted ? '•' : '';

  const copy = async (text: string, message: string) => {
    try {
      await navigator.clipboard.writeText(text);
      showToast(message, 'success');
    } catch {
      showToast('Clipboard is unavailable in this browser', 'error');
    }
  };

  return (
    <>
      <div
        role="tablist"
        aria-label="Problem panels"
        style={{
          display: 'flex',
          gap: 2,
          padding: '0 12px',
          borderBottom: '1px solid var(--border)',
          flexShrink: 0,
          overflowX: 'auto',
        }}
      >
        {TABS.map((item) => {
          const active = tab === item.key;
          return (
            <button
              key={item.key}
              type="button"
              role="tab"
              aria-selected={active}
              onClick={() => onTabChange(item.key)}
              className="ac-hover-text"
              style={{
                height: 42,
                padding: '0 12px',
                border: 'none',
                borderBottom: `2px solid ${active ? 'var(--accent)' : 'transparent'}`,
                background: 'none',
                cursor: 'pointer',
                fontSize: 12.5,
                fontWeight: active ? 600 : 500,
                color: active ? 'var(--text)' : 'var(--text3)',
                whiteSpace: 'nowrap',
              }}
            >
              {item.label}
            </button>
          );
        })}
      </div>

      <div className="ac-problem-panel-content" style={{ flex: 1, overflowY: 'auto' }}>
        {tab === 'description' && (
          <>
            <div
              className="ac-problem-statement-header"
              style={{
                position: 'sticky',
                top: 0,
                zIndex: 5,
                display: 'flex',
                flexWrap: 'wrap',
                alignItems: 'center',
                gap: 9,
                background: 'var(--surface)',
                borderBottom: '1px solid var(--border)',
              }}
            >
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11.5, color: 'var(--text3)' }}>
                #{problem.id}
              </span>
              <h1 style={{ margin: 0, fontSize: 16, fontWeight: 650, letterSpacing: '-0.01em' }}>
                {problem.title}
              </h1>
              <span
                style={{
                  fontSize: 11,
                  fontWeight: 600,
                  color: diff.color,
                  background: diff.bg,
                  borderRadius: 6,
                  padding: '2px 9px',
                }}
              >
                {diff.label}
              </span>
              <span
                role="img"
                aria-label={status}
                title={status}
                style={{
                  width: 20,
                  height: 20,
                  borderRadius: '50%',
                  border: `1.5px solid ${statusColor}`,
                  color: statusColor,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 12,
                  fontWeight: 700,
                  flexShrink: 0,
                }}
              >
                {statusIcon}
              </span>
              <span style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: 5 }}>
                <span
                  style={{
                    fontSize: 11.5,
                    color: 'var(--text3)',
                    whiteSpace: 'nowrap',
                    marginRight: 3,
                    fontFamily: 'var(--font-mono)',
                  }}
                >
                  {formatTimeLimit(problem.time_limit)} · {formatMemoryLimit(problem.memory_limit)}
                </span>
                <button
                  type="button"
                  onClick={() =>
                    copy(`${window.location.origin}/problems/${problem.slug}`, 'Problem link copied')
                  }
                  aria-label="Copy problem link"
                  title="Copy problem link"
                  className="ac-hover-surface2"
                  style={{
                    width: 28,
                    height: 28,
                    border: '1px solid var(--border)',
                    borderRadius: 7,
                    background: 'var(--surface)',
                    color: 'var(--text2)',
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                  }}
                >
                  <Icon.Link />
                </button>
              </span>
            </div>

            <RichText text={problem.description} />

            {problem.input_format && (
              <section aria-labelledby="problem-input-format">
                <h2 id="problem-input-format" className="ac-statement-section-heading">Input</h2>
                <RichText text={problem.input_format} />
              </section>
            )}

            {problem.output_format && (
              <section aria-labelledby="problem-output-format">
                <h2 id="problem-output-format" className="ac-statement-section-heading">Output</h2>
                <RichText text={problem.output_format} />
              </section>
            )}

            {problem.constraints && problem.constraints.length > 0 && (
              <>
                <h2 className="ac-statement-section-heading">Constraints</h2>
                <ul
                  className="ac-problem-constraints"
                  style={{
                    fontFamily: 'var(--font-mono)',
                  }}
                >
                  {problem.constraints.map((constraint, index) => (
                    <li key={index}>{constraint}</li>
                  ))}
                </ul>
              </>
            )}

            {problem.examples && problem.examples.length > 0 && (
              <section className="ac-problem-examples" aria-labelledby="problem-examples">
                <h2 id="problem-examples" className="ac-statement-section-heading">Examples</h2>
                {problem.examples.map((example, index) => (
                  <article
                    key={index}
                    className="ac-problem-example"
                  >
                    <div
                      className="ac-problem-example-header"
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 8,
                      }}
                    >
                      <h3>Example {index + 1}</h3>
                    </div>
                    <div className="ac-problem-example-data">
                      <section className="ac-problem-example-code" aria-label={`Example ${index + 1} input`}>
                        <div className="ac-problem-example-code-header">
                          <h4>Input</h4>
                          <span className="ac-problem-example-actions">
                            <button
                              type="button"
                              onClick={() => copy(example.input, 'Input copied')}
                              aria-label={`Copy input for example ${index + 1}`}
                              title="Copy input"
                              className="ac-problem-example-icon-button ac-hover-surface2"
                            >
                              <Icon.Copy />
                            </button>
                            <button
                              type="button"
                              onClick={() => onUseExample(example.input, example.expected_output)}
                              aria-label={`Use input from example ${index + 1} as a test`}
                              title="Use as test"
                              className="ac-problem-example-icon-button ac-hover-surface2"
                            >
                              <Icon.Play />
                            </button>
                          </span>
                        </div>
                        <pre><code>{example.input}</code></pre>
                      </section>
                      <section className="ac-problem-example-code" aria-label={`Example ${index + 1} output`}>
                        <div className="ac-problem-example-code-header">
                          <h4>Output</h4>
                          <span className="ac-problem-example-actions">
                            <button
                              type="button"
                              onClick={() => copy(example.expected_output, 'Output copied')}
                              aria-label={`Copy output for example ${index + 1}`}
                              title="Copy output"
                              className="ac-problem-example-icon-button ac-hover-surface2"
                            >
                              <Icon.Copy />
                            </button>
                          </span>
                        </div>
                        <pre><code>{example.expected_output}</code></pre>
                      </section>
                    </div>
                    {example.explanation?.trim() && (
                      <section className="ac-problem-example-explanation" aria-label={`Example ${index + 1} explanation`}>
                        <h4>Explanation</h4>
                        <p>{example.explanation}</p>
                      </section>
                    )}
                  </article>
                ))}
              </section>
            )}

            {problem.hints && problem.hints.length > 0 && <HintList hints={problem.hints} />}

            {problem.tags && problem.tags.length > 0 && (
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, margin: '18px 0' }}>
                {problem.tags.map((tag) => (
                  <Link
                    key={tag.id}
                    href={`/problems?tag_slug=${encodeURIComponent(tag.slug)}`}
                    style={{
                      fontSize: 11,
                      color: 'var(--text2)',
                      background: 'var(--surface2)',
                      border: '1px solid var(--border)',
                      borderRadius: 6,
                      padding: '3px 9px',
                      textDecoration: 'none',
                    }}
                  >
                    {tag.name}
                  </Link>
                ))}
              </div>
            )}

          </>
        )}

        {tab === 'submissions' && (
          <ProblemSubmissions
            problemId={problem.id}
            signedIn={signedIn}
            refreshKey={historyRefreshKey}
            onUseSubmissionCode={onUseSubmissionCode}
          />
        )}

        {tab === 'solutions' && (
          <EmptyState
            title="Editorials are not available yet"
            description="The problem service does not expose editorial content."
            nodes={4}
            done={2}
          />
        )}

        {tab === 'discussion' && (
          <EmptyState
            title="Discussions are not wired up"
            description="No discussion endpoint exists on the gateway yet."
            nodes={4}
            done={1}
          />
        )}
      </div>
    </>
  );
}

function HintList({ hints }: { hints: string[] }) {
  const [open, setOpen] = useState<number | null>(null);
  return (
    <>
      <h2 className="ac-statement-section-heading">Hints</h2>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginBottom: 18 }}>
        {hints.map((hint, index) => (
          <div
            key={index}
            style={{ border: '1px solid var(--border)', borderRadius: 10, overflow: 'hidden' }}
          >
            <button
              type="button"
              onClick={() => setOpen(open === index ? null : index)}
              aria-expanded={open === index}
              className="ac-hover-surface2"
              style={{
                display: 'flex',
                width: '100%',
                alignItems: 'center',
                gap: 8,
                padding: '9px 12px',
                border: 'none',
                background: 'var(--surface2)',
                cursor: 'pointer',
                fontSize: 12.5,
                fontWeight: 600,
                color: 'var(--text2)',
                textAlign: 'left',
              }}
            >
              Hint {index + 1}
              <span style={{ marginLeft: 'auto', display: 'flex' }}>
                <Icon.Chevron color="var(--text3)" />
              </span>
            </button>
            {open === index && (
              <p
                style={{
                  margin: 0,
                  padding: '10px 12px',
                  fontSize: 13,
                  lineHeight: 1.6,
                  color: 'var(--text2)',
                }}
              >
                {hint}
              </p>
            )}
          </div>
        ))}
      </div>
    </>
  );
}

const SUBMISSIONS_PAGE_SIZE = 10;
const DETAIL_POLL_INTERVAL_MS = 1500;

function ProblemSubmissions({
  problemId,
  signedIn,
  refreshKey,
  onUseSubmissionCode,
}: {
  problemId: number;
  signedIn: boolean;
  refreshKey: number;
  onUseSubmissionCode: (submission: SubmissionDetail) => void;
}) {
  const [items, setItems] = useState<SubmissionListItem[] | null>(null);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(0);
  const [failed, setFailed] = useState('');
  const [retryKey, setRetryKey] = useState(0);
  const [selectedID, setSelectedID] = useState<number | null>(null);

  useEffect(() => {
    setPage(1);
    setSelectedID(null);
  }, [problemId, refreshKey]);

  useEffect(() => {
    if (!signedIn) {
      setItems(null);
      setFailed('');
      return;
    }
    const controller = new AbortController();
    setItems(null);
    setFailed('');
    submissionApi
      .listMine({ problem_id: problemId, page, limit: SUBMISSIONS_PAGE_SIZE }, controller.signal)
      .then((res) => {
        setItems(res.items ?? []);
        setTotalPages(res.pagination.total_pages ?? 0);
      })
      .catch((err) => {
        if (controller.signal.aborted) return;
        if (err instanceof NetworkError) setFailed('AstraCode is temporarily unreachable. Check your connection and try again.');
        else if (err instanceof ApiError) setFailed(err.message || 'Submission history request failed.');
        else setFailed('Submission history request failed.');
      });
    return () => controller.abort();
  }, [page, problemId, refreshKey, retryKey, signedIn]);

  const updateListItem = useCallback((detail: SubmissionDetail) => {
    setItems((current) => {
      if (!current) return current;
      return current.map((item) =>
        item.id === detail.id
          ? {
              ...item,
              status: detail.status,
              execution_time_ms: detail.execution_time_ms,
              memory_used_kb: detail.memory_used_kb,
              passed_testcases: detail.passed_testcases,
              total_testcases: detail.total_testcases,
            }
          : item,
      );
    });
  }, []);

  if (!signedIn) {
    return (
      <EmptyState
        title="Sign in to see your submissions"
        description="Your judge history for this problem lives behind authentication."
      />
    );
  }
  if (failed) {
    return (
      <EmptyState
        title="Couldn't load your submissions"
        description={failed}
        action={
          <button
            type="button"
            onClick={() => setRetryKey((current) => current + 1)}
            className="ac-hover-accent"
            style={primaryButton}
          >
            Retry
          </button>
        }
      />
    );
  }
  if (!items) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        <SkeletonBar height={34} radius={8} />
        <SkeletonBar height={34} radius={8} />
        <SkeletonBar height={34} radius={8} />
      </div>
    );
  }
  if (items.length === 0) {
    return (
      <EmptyState
        title="No submissions on this problem yet"
        description="Submit a solution and it will show up here."
      />
    );
  }

  return (
    <>
      <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 8, marginBottom: 12 }}>
        <h2 style={{ margin: 0, fontSize: 14, fontWeight: 650 }}>Your submissions on this problem</h2>
        <span style={{ marginLeft: 'auto', fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text3)' }}>
          Page {page}{totalPages > 0 ? ` / ${totalPages}` : ''}
        </span>
      </div>
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          border: '1px solid var(--border)',
          borderRadius: 10,
          overflow: 'hidden',
        }}
      >
        {items.map((submission) => {
          const verdict = verdictMeta(submission.status);
          return (
            <button
              key={submission.id}
              type="button"
              onClick={() => setSelectedID((current) => (current === submission.id ? null : submission.id))}
              className="ac-hover-surface2"
              style={{
                display: 'flex',
                flexWrap: 'wrap',
                alignItems: 'center',
                gap: 10,
                padding: '10px 12px',
                borderTop: '1px solid var(--border)',
                borderRight: 'none',
                borderBottom: 'none',
                borderLeft: 'none',
                marginTop: -1,
                background: selectedID === submission.id ? 'var(--accent-soft)' : 'var(--surface)',
                color: 'inherit',
                cursor: 'pointer',
                textAlign: 'left',
              }}
            >
              <span
                aria-hidden="true"
                style={{
                  width: 22,
                  height: 22,
                  borderRadius: 6,
                  background: verdict.bg,
                  color: verdict.color,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 11,
                  fontWeight: 700,
                  flexShrink: 0,
                }}
              >
                {verdict.icon}
              </span>
              <span style={{ flex: 1, minWidth: 0 }}>
                <span
                  style={{ display: 'block', fontSize: 12.5, fontWeight: 600, color: verdict.color }}
                >
                  {verdict.label}
                </span>
                <span style={{ fontSize: 11, color: 'var(--text3)' }}>
                  {languageLabel(submission.language)} · {formatRuntimeMs(submission.execution_time_ms)} ·{' '}
                  {formatMemoryKb(submission.memory_used_kb)}
                </span>
              </span>
              <span style={submissionMetric}>
                {formatTestcaseCount(submission.passed_testcases, submission.total_testcases)}
                <small style={submissionMetricLabel}>tests</small>
              </span>
              <span style={submissionMetric}>
                {timeAgo(submission.created_at)}
                <small style={submissionMetricLabel}>submitted</small>
              </span>
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text3)' }}>
                #{submission.id}
              </span>
            </button>
          );
        })}
      </div>
      {totalPages > 1 && (
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 8, marginTop: 12 }}>
          <button
            type="button"
            onClick={() => setPage((current) => Math.max(1, current - 1))}
            disabled={page <= 1}
            className="ac-hover-surface2"
            style={pagerButton(page <= 1)}
          >
            Previous
          </button>
          <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text3)' }}>
            {items.length} shown
          </span>
          <button
            type="button"
            onClick={() => setPage((current) => Math.min(totalPages, current + 1))}
            disabled={page >= totalPages}
            className="ac-hover-surface2"
            style={pagerButton(page >= totalPages)}
          >
            Next
          </button>
        </div>
      )}

      {selectedID !== null && (
        <SubmissionDetailPanel
          submissionID={selectedID}
          onUseSubmissionCode={onUseSubmissionCode}
          onClose={() => setSelectedID(null)}
          onUpdateListItem={updateListItem}
        />
      )}
    </>
  );
}

function SubmissionDetailPanel({
  submissionID,
  onUseSubmissionCode,
  onUpdateListItem,
  onClose,
}: {
  submissionID: number;
  onUseSubmissionCode: (submission: SubmissionDetail) => void;
  onUpdateListItem: (submission: SubmissionDetail) => void;
  onClose: () => void;
}) {
  const { showToast } = useToast();
  const [detail, setDetail] = useState<SubmissionDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [failed, setFailed] = useState('');
  const [retryKey, setRetryKey] = useState(0);

  useEffect(() => {
    const controller = new AbortController();
    let timeout: ReturnType<typeof setTimeout> | null = null;

    const load = async () => {
      setLoading(true);
      setFailed('');
      try {
        const next = await submissionApi.get(submissionID, controller.signal);
        setDetail(next);
        onUpdateListItem(next);
        if (isPendingStatus(next.status) && !controller.signal.aborted) {
          timeout = setTimeout(load, DETAIL_POLL_INTERVAL_MS);
        }
      } catch (err) {
        if (controller.signal.aborted) return;
        if (err instanceof NetworkError) setFailed('AstraCode is temporarily unreachable. Check your connection and try again.');
        else if (err instanceof ApiError) setFailed(err.message || 'Submission detail request failed.');
        else setFailed('Submission detail request failed.');
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    };

    void load();
    return () => {
      controller.abort();
      if (timeout) clearTimeout(timeout);
    };
  }, [onUpdateListItem, retryKey, submissionID]);

  return (
    <section
      aria-label="Submission detail"
      style={{
        marginTop: 14,
        border: '1px solid var(--border)',
        borderRadius: 12,
        background: 'var(--surface)',
        overflow: 'hidden',
      }}
    >
      <div
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          alignItems: 'center',
          gap: 8,
          padding: '10px 12px',
          borderBottom: '1px solid var(--border)',
          background: 'var(--surface2)',
        }}
      >
        <h3 style={{ margin: 0, fontSize: 13, fontWeight: 700 }}>Submission #{submissionID}</h3>
        {loading && <Spinner size={12} />}
        <button
          type="button"
          onClick={onClose}
          aria-label="Close submission detail"
          className="ac-hover-text"
          style={{
            marginLeft: 'auto',
            border: 'none',
            background: 'none',
            color: 'var(--text3)',
            cursor: 'pointer',
            fontSize: 13,
          }}
        >
          ✕
        </button>
      </div>

      <div style={{ padding: 12 }}>
        {failed && (
          <div role="alert" style={{ display: 'grid', gap: 10 }}>
            <p style={{ margin: 0, fontSize: 12.5, color: 'var(--error)' }}>{failed}</p>
            <button
              type="button"
              onClick={() => setRetryKey((current) => current + 1)}
              className="ac-hover-surface2"
              style={smallButton}
            >
              Retry
            </button>
          </div>
        )}

        {!failed && !detail && (
          <div style={{ display: 'grid', gap: 8 }}>
            <SkeletonBar height={18} radius={6} />
            <SkeletonBar height={18} radius={6} width="72%" />
            <SkeletonBar height={90} radius={8} />
          </div>
        )}

        {detail && (
          <SubmissionDetailContent
            detail={detail}
            onUseSubmissionCode={onUseSubmissionCode}
            onCopy={async (text, message) => {
              try {
                await navigator.clipboard.writeText(text);
                showToast(message, 'success');
              } catch {
                showToast('Clipboard is unavailable in this browser', 'error');
              }
            }}
          />
        )}
      </div>
    </section>
  );
}

function SubmissionDetailContent({
  detail,
  onUseSubmissionCode,
  onCopy,
}: {
  detail: SubmissionDetail;
  onUseSubmissionCode: (submission: SubmissionDetail) => void;
  onCopy: (text: string, message: string) => void;
}) {
  const verdict = verdictMeta(detail.status);
  const output = submissionDetailOutput(detail);

  return (
    <div style={{ display: 'grid', gap: 12 }}>
      <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 8 }}>
        <span
          aria-hidden="true"
          style={{
            width: 22,
            height: 22,
            borderRadius: verdict.shape,
            background: verdict.bg,
            color: verdict.color,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 12,
            fontWeight: 800,
          }}
        >
          {verdict.icon}
        </span>
        <strong style={{ color: verdict.color, fontSize: 13 }}>{verdict.label}</strong>
        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text3)' }}>
          {formatDateTime(detail.created_at)}
        </span>
      </div>

      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit,minmax(120px,1fr))',
          gap: 8,
        }}
      >
        <ResultMeta value={languageLabel(detail.language)} label="Language" />
        <ResultMeta value={formatRuntimeMs(detail.execution_time_ms)} label="Runtime" />
        <ResultMeta value={formatMemoryKb(detail.memory_used_kb)} label="Memory" />
        <ResultMeta
          value={formatTestcaseCount(detail.passed_testcases, detail.total_testcases)}
          label="Test cases"
        />
      </div>

      {output && (
        <div>
          <div style={detailBlockLabel}>{output.label}</div>
          <pre style={detailOutputBlock}>{output.output}</pre>
        </div>
      )}

      <div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
          <div style={detailBlockLabel}>Source code</div>
          <button
            type="button"
            onClick={() => onCopy(detail.source_code, 'Source code copied')}
            className="ac-hover-surface2-text"
            style={{ ...smallButton, marginLeft: 'auto' }}
          >
            Copy
          </button>
          <button
            type="button"
            onClick={() => onUseSubmissionCode(detail)}
            className="ac-hover-accent"
            style={primaryButton}
          >
            Use this code
          </button>
        </div>
        <pre style={sourceCodeBlock}>{detail.source_code}</pre>
      </div>
    </div>
  );
}

function submissionDetailOutput(
  detail: Pick<SubmissionDetail, 'status' | 'compile_output' | 'error_message'>,
): { label: string; output: string } | null {
  if (detail.status === 'COMPILATION_ERROR' && detail.compile_output) {
    return { label: 'Compilation output', output: detail.compile_output };
  }
  if (detail.status === 'RUNTIME_ERROR' && detail.error_message) {
    return { label: 'Runtime error', output: detail.error_message };
  }
  if (detail.status === 'SYSTEM_ERROR' && detail.error_message) {
    return { label: 'System error', output: detail.error_message };
  }
  if (
    (detail.status === 'TIME_LIMIT_EXCEEDED' || detail.status === 'MEMORY_LIMIT_EXCEEDED') &&
    detail.error_message
  ) {
    return { label: 'Message', output: detail.error_message };
  }
  if (detail.status === 'OUTPUT_LIMIT_EXCEEDED' && detail.error_message) {
    return { label: 'Message', output: detail.error_message };
  }
  return null;
}

function ResultMeta({ value, label }: { value: string; label: string }) {
  return (
    <div style={{ border: '1px solid var(--border)', borderRadius: 8, padding: '8px 10px' }}>
      <span style={{ display: 'block', fontFamily: 'var(--font-mono)', fontSize: 12.5, fontWeight: 650 }}>
        {value}
      </span>
      <span style={{ display: 'block', marginTop: 2, fontSize: 10.5, color: 'var(--text3)' }}>
        {label}
      </span>
    </div>
  );
}

const submissionMetric: React.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'flex-end',
  gap: 1,
  minWidth: 74,
  fontFamily: 'var(--font-mono)',
  fontSize: 11.5,
  color: 'var(--text2)',
};

const submissionMetricLabel: React.CSSProperties = {
  fontFamily: 'var(--font-sans)',
  fontSize: 10,
  color: 'var(--text3)',
};

function pagerButton(disabled: boolean): React.CSSProperties {
  return {
    height: 32,
    padding: '0 12px',
    border: '1px solid var(--border)',
    borderRadius: 8,
    background: 'var(--surface)',
    color: 'var(--text2)',
    fontSize: 12,
    fontWeight: 600,
    cursor: disabled ? 'not-allowed' : 'pointer',
    opacity: disabled ? 0.45 : 1,
  };
}

const primaryButton: React.CSSProperties = {
  height: 28,
  padding: '0 10px',
  border: 'none',
  borderRadius: 7,
  background: 'var(--accent)',
  color: 'var(--accent-ink)',
  fontSize: 11,
  fontWeight: 650,
  cursor: 'pointer',
  whiteSpace: 'nowrap',
};

const detailBlockLabel: React.CSSProperties = {
  marginBottom: 6,
  fontSize: 11,
  fontWeight: 700,
  letterSpacing: '.06em',
  textTransform: 'uppercase',
  color: 'var(--text3)',
};

const detailOutputBlock: React.CSSProperties = {
  margin: 0,
  padding: 10,
  border: '1px solid var(--border)',
  borderRadius: 8,
  background: 'var(--code-bg)',
  color: 'var(--error)',
  fontFamily: 'var(--font-mono)',
  fontSize: 11.5,
  lineHeight: 1.6,
  whiteSpace: 'pre-wrap',
  overflowX: 'auto',
};

const sourceCodeBlock: React.CSSProperties = {
  margin: 0,
  maxHeight: 260,
  overflow: 'auto',
  padding: 12,
  border: '1px solid var(--border)',
  borderRadius: 8,
  background: 'var(--code-bg)',
  color: 'var(--code-fg)',
  fontFamily: 'var(--font-mono)',
  fontSize: 11.5,
  lineHeight: 1.65,
  whiteSpace: 'pre',
};

const smallButton: React.CSSProperties = {
  height: 26,
  padding: '0 9px',
  border: '1px solid var(--border)',
  borderRadius: 6,
  background: 'var(--surface)',
  color: 'var(--text2)',
  fontSize: 10.5,
  fontWeight: 600,
  cursor: 'pointer',
  whiteSpace: 'nowrap',
};
