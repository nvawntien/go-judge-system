'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import { RichText } from '@/components/RichText';
import { EmptyState, Icon, SkeletonBar } from '@/components/ui';
import { useToast } from '@/components/ToastProvider';
import { submissionApi } from '@/lib/api';
import {
  difficultyMeta,
  formatMemoryLimit,
  formatTimeLimit,
  languageLabel,
  timeAgo,
  verdictMeta,
} from '@/lib/format';
import type { Problem, SubmissionListItem } from '@/lib/types';

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
}: {
  problem: Problem;
  tab: ProblemTab;
  onTabChange: (tab: ProblemTab) => void;
  solved: boolean;
  attempted: boolean;
  signedIn: boolean;
  onUseExample: (input: string, expected: string) => void;
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

      <div style={{ flex: 1, overflowY: 'auto', padding: '20px 22px 32px' }}>
        {tab === 'description' && (
          <>
            <div
              style={{
                position: 'sticky',
                top: 0,
                zIndex: 5,
                display: 'flex',
                flexWrap: 'wrap',
                alignItems: 'center',
                gap: 9,
                margin: '-20px -22px 18px',
                padding: '11px 22px',
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

            {problem.constraints && problem.constraints.length > 0 && (
              <>
                <h2 style={sectionHeading}>Constraints</h2>
                <ul
                  style={{
                    margin: '0 0 20px',
                    paddingLeft: 20,
                    fontFamily: 'var(--font-mono)',
                    fontSize: 12,
                    lineHeight: 1.9,
                    color: 'var(--text2)',
                  }}
                >
                  {problem.constraints.map((constraint, index) => (
                    <li key={index}>{constraint}</li>
                  ))}
                </ul>
              </>
            )}

            {problem.examples && problem.examples.length > 0 && (
              <>
                <h2 style={sectionHeading}>Examples</h2>
                {problem.examples.map((example, index) => (
                  <div
                    key={index}
                    style={{
                      border: '1px solid var(--border)',
                      borderRadius: 10,
                      marginBottom: 12,
                      overflow: 'hidden',
                    }}
                  >
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 8,
                        padding: '6px 8px 6px 12px',
                        background: 'var(--surface2)',
                        borderBottom: '1px solid var(--border)',
                      }}
                    >
                      <svg width="30" height="10" viewBox="0 0 30 10" aria-hidden="true">
                        <path d="M4 5 H26" stroke="var(--accent-soft2)" strokeWidth="1.4" />
                        <circle cx="4" cy="5" r="2.6" fill="var(--accent)" />
                        <circle cx="15" cy="5" r="2.6" fill="var(--accent-soft2)" />
                        <circle cx="26" cy="5" r="2.6" fill="var(--accent-soft2)" />
                      </svg>
                      <span style={{ fontSize: 11, fontWeight: 650, color: 'var(--text2)' }}>
                        Example {index + 1}
                      </span>
                      <span style={{ marginLeft: 'auto', display: 'flex', gap: 6 }}>
                        <button
                          type="button"
                          onClick={() => copy(example.input, 'Input copied')}
                          className="ac-hover-surface2-text"
                          style={smallButton}
                        >
                          Copy input
                        </button>
                        <button
                          type="button"
                          onClick={() => onUseExample(example.input, example.output)}
                          className="ac-hover-surface2-text"
                          style={smallButton}
                        >
                          Use as test
                        </button>
                      </span>
                    </div>
                    <div
                      style={{
                        padding: '10px 12px',
                        fontFamily: 'var(--font-mono)',
                        fontSize: 12,
                        lineHeight: 1.7,
                        background: 'var(--code-bg)',
                        display: 'flex',
                        flexDirection: 'column',
                        gap: 6,
                      }}
                    >
                      <div style={{ display: 'flex', gap: 10 }}>
                        <span style={{ color: 'var(--text3)', flexShrink: 0, width: 52 }}>Input</span>
                        <span style={{ color: 'var(--code-fg)', whiteSpace: 'pre-wrap' }}>
                          {example.input}
                        </span>
                      </div>
                      <div style={{ display: 'flex', gap: 10 }}>
                        <span style={{ color: 'var(--text3)', flexShrink: 0, width: 52 }}>Output</span>
                        <span style={{ color: 'var(--syn-str)', whiteSpace: 'pre-wrap' }}>
                          {example.output}
                        </span>
                      </div>
                      {example.explanation && (
                        <div style={{ color: 'var(--syn-com)', whiteSpace: 'pre-wrap' }}>
                          // {example.explanation}
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </>
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

            <div
              style={{
                display: 'flex',
                gap: 16,
                flexWrap: 'wrap',
                paddingTop: 14,
                borderTop: '1px solid var(--border)',
                fontSize: 12,
                color: 'var(--text3)',
              }}
            >
              <span>
                Time limit <strong style={{ color: 'var(--text2)' }}>{formatTimeLimit(problem.time_limit)}</strong>
              </span>
              <span>
                Memory limit{' '}
                <strong style={{ color: 'var(--text2)' }}>{formatMemoryLimit(problem.memory_limit)}</strong>
              </span>
              <span>
                Slug <strong style={{ color: 'var(--text2)', fontFamily: 'var(--font-mono)' }}>{problem.slug}</strong>
              </span>
            </div>
          </>
        )}

        {tab === 'submissions' && (
          <ProblemSubmissions problemId={problem.id} signedIn={signedIn} />
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
      <h2 style={sectionHeading}>Hints</h2>
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

function ProblemSubmissions({ problemId, signedIn }: { problemId: number; signedIn: boolean }) {
  const [items, setItems] = useState<SubmissionListItem[] | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    if (!signedIn) return;
    const controller = new AbortController();
    submissionApi
      .listMine({ problem_id: problemId, limit: 20 }, controller.signal)
      .then((res) => setItems(res.items ?? []))
      .catch(() => {
        if (!controller.signal.aborted) setFailed(true);
      });
    return () => controller.abort();
  }, [problemId, signedIn]);

  if (!signedIn) {
    return (
      <EmptyState
        title="Sign in to see your submissions"
        description="Your judge history for this problem lives behind authentication."
      />
    );
  }
  if (failed) return <EmptyState title="Couldn't load your submissions" />;
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
      <h2 style={{ margin: '0 0 12px', fontSize: 14, fontWeight: 650 }}>
        Your submissions on this problem
      </h2>
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
            <div
              key={submission.id}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 10,
                padding: '10px 12px',
                borderTop: '1px solid var(--border)',
                marginTop: -1,
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
                  {languageLabel(submission.language)} · {timeAgo(submission.created_at)}
                </span>
              </span>
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text3)' }}>
                #{submission.id}
              </span>
            </div>
          );
        })}
      </div>
    </>
  );
}

const sectionHeading: React.CSSProperties = {
  margin: '18px 0 8px',
  fontSize: 12,
  fontWeight: 650,
  letterSpacing: '.07em',
  textTransform: 'uppercase',
  color: 'var(--text3)',
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
