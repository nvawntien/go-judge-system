import type { Difficulty, LanguageCode, SubmissionStatus } from './types';

/* ---------------------------------------------------------------- verdict */

export interface VerdictMeta {
  label: string;
  /** Short glyph used in the round status chip. */
  icon: string;
  color: string;
  bg: string;
  /** Square badge for failures, circle for passes — never colour alone. */
  shape: string;
  terminal: boolean;
}

const VERDICTS: Record<SubmissionStatus, VerdictMeta> = {
  PENDING: {
    label: 'Queued',
    icon: '◷',
    color: 'var(--text2)',
    bg: 'var(--surface3)',
    shape: '50%',
    terminal: false,
  },
  JUDGING: {
    label: 'Judging',
    icon: '◷',
    color: 'var(--accent-fg)',
    bg: 'var(--accent-soft)',
    shape: '50%',
    terminal: false,
  },
  ACCEPTED: {
    label: 'Accepted',
    icon: '✓',
    color: 'var(--success)',
    bg: 'var(--success-bg)',
    shape: '50%',
    terminal: true,
  },
  WRONG_ANSWER: {
    label: 'Wrong Answer',
    icon: '✕',
    color: 'var(--error)',
    bg: 'var(--error-bg)',
    shape: '5px',
    terminal: true,
  },
  TIME_LIMIT_EXCEEDED: {
    label: 'Time Limit Exceeded',
    icon: '◷',
    color: 'var(--warn)',
    bg: 'var(--warn-bg)',
    shape: '5px',
    terminal: true,
  },
  MEMORY_LIMIT_EXCEEDED: {
    label: 'Memory Limit Exceeded',
    icon: '▣',
    color: 'var(--warn)',
    bg: 'var(--warn-bg)',
    shape: '5px',
    terminal: true,
  },
  OUTPUT_LIMIT_EXCEEDED: {
    label: 'Output Limit Exceeded',
    icon: '▣',
    color: 'var(--warn)',
    bg: 'var(--warn-bg)',
    shape: '5px',
    terminal: true,
  },
  RUNTIME_ERROR: {
    label: 'Runtime Error',
    icon: '!',
    color: 'var(--error)',
    bg: 'var(--error-bg)',
    shape: '5px',
    terminal: true,
  },
  COMPILATION_ERROR: {
    label: 'Compilation Error',
    icon: '!',
    color: 'var(--error)',
    bg: 'var(--error-bg)',
    shape: '5px',
    terminal: true,
  },
  SYSTEM_ERROR: {
    label: 'System Error',
    icon: '!',
    color: 'var(--error)',
    bg: 'var(--error-bg)',
    shape: '5px',
    terminal: true,
  },
};

const UNKNOWN_VERDICT: VerdictMeta = {
  label: 'Unknown',
  icon: '?',
  color: 'var(--text2)',
  bg: 'var(--surface3)',
  shape: '5px',
  terminal: true,
};

export const TERMINAL_SUBMISSION_STATUSES = new Set<SubmissionStatus>([
  'ACCEPTED',
  'WRONG_ANSWER',
  'TIME_LIMIT_EXCEEDED',
  'MEMORY_LIMIT_EXCEEDED',
  'OUTPUT_LIMIT_EXCEEDED',
  'RUNTIME_ERROR',
  'COMPILATION_ERROR',
  'SYSTEM_ERROR',
]);

export const NON_TERMINAL_SUBMISSION_STATUSES = new Set<SubmissionStatus>(['PENDING', 'JUDGING']);

export function verdictMeta(status: string): VerdictMeta {
  return VERDICTS[status as SubmissionStatus] ?? UNKNOWN_VERDICT;
}

export function isPendingStatus(status: string): boolean {
  return NON_TERMINAL_SUBMISSION_STATUSES.has(status as SubmissionStatus);
}

export function isTerminalSubmissionStatus(status: string): boolean {
  return TERMINAL_SUBMISSION_STATUSES.has(status as SubmissionStatus);
}

/** Options for the submissions status filter, in the order the design lists them. */
export const STATUS_FILTERS: { value: SubmissionStatus | ''; label: string }[] = [
  { value: '', label: 'Status: All' },
  { value: 'ACCEPTED', label: 'Accepted' },
  { value: 'WRONG_ANSWER', label: 'Wrong Answer' },
  { value: 'TIME_LIMIT_EXCEEDED', label: 'Time Limit Exceeded' },
  { value: 'MEMORY_LIMIT_EXCEEDED', label: 'Memory Limit Exceeded' },
  { value: 'OUTPUT_LIMIT_EXCEEDED', label: 'Output Limit Exceeded' },
  { value: 'RUNTIME_ERROR', label: 'Runtime Error' },
  { value: 'COMPILATION_ERROR', label: 'Compilation Error' },
  { value: 'SYSTEM_ERROR', label: 'System Error' },
  { value: 'PENDING', label: 'Queued' },
  { value: 'JUDGING', label: 'Judging' },
];

/* ------------------------------------------------------------- difficulty */

export interface DifficultyMeta {
  label: string;
  color: string;
  bg: string;
}

export function difficultyMeta(difficulty: string): DifficultyMeta {
  switch (difficulty?.toLowerCase()) {
    case 'easy':
      return { label: 'Easy', color: 'var(--easy)', bg: 'var(--success-bg)' };
    case 'hard':
      return { label: 'Hard', color: 'var(--hard)', bg: 'var(--error-bg)' };
    case 'medium':
    default:
      return { label: 'Medium', color: 'var(--med)', bg: 'var(--warn-bg)' };
  }
}

export const DIFFICULTIES: Difficulty[] = ['easy', 'medium', 'hard'];

/* --------------------------------------------------------------- language */

export interface LanguageMeta {
  code: LanguageCode;
  label: string;
  /** main.<ext> shown next to the language picker. */
  ext: string;
  /** Colour used in the profile language-usage bar. */
  color: string;
}

export const LANGUAGES: LanguageMeta[] = [
  { code: 'GO', label: 'Go', ext: 'go', color: 'var(--accent)' },
  { code: 'CPP', label: 'C++', ext: 'cpp', color: 'var(--syn-type)' },
  { code: 'PYTHON', label: 'Python', ext: 'py', color: 'var(--syn-num)' },
  { code: 'JAVA', label: 'Java', ext: 'java', color: 'var(--syn-fn)' },
];

export function languageMeta(code: string): LanguageMeta {
  return (
    LANGUAGES.find((l) => l.code === code?.toUpperCase()) ?? {
      code: 'GO',
      label: code || 'Unknown',
      ext: 'txt',
      color: 'var(--text3)',
    }
  );
}

export function languageLabel(code: string): string {
  return languageMeta(code).label;
}

/* ------------------------------------------------------------------- time */

const UNITS: [number, Intl.RelativeTimeFormatUnit][] = [
  [60, 'second'],
  [60, 'minute'],
  [24, 'hour'],
  [7, 'day'],
  [4.34524, 'week'],
  [12, 'month'],
  [Number.POSITIVE_INFINITY, 'year'],
];

/** "2 hours ago" / "yesterday" — matches the prototype's relative wording. */
export function timeAgo(input: string | number | Date): string {
  const date = input instanceof Date ? input : new Date(input);
  const seconds = (Date.now() - date.getTime()) / 1000;
  if (!Number.isFinite(seconds)) return '';
  if (seconds < 45) return 'just now';

  let value = seconds;
  for (const [step, unit] of UNITS) {
    if (Math.abs(value) < step) {
      const rounded = Math.round(value);
      if (unit === 'day' && rounded === 1) return 'yesterday';
      return new Intl.RelativeTimeFormat('en', { numeric: 'auto' }).format(-rounded, unit);
    }
    value /= step;
  }
  return date.toLocaleDateString();
}

export function formatDate(input: string | Date): string {
  const date = input instanceof Date ? input : new Date(input);
  if (Number.isNaN(date.getTime())) return '—';
  return date.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
}

export function formatDateTime(input: string | Date): string {
  const date = input instanceof Date ? input : new Date(input);
  if (Number.isNaN(date.getTime())) return '—';
  return date.toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

/** Local YYYY-MM-DD key, used to bucket submissions per day. */
export function dayKey(input: string | Date): string {
  const date = input instanceof Date ? input : new Date(input);
  const month = `${date.getMonth() + 1}`.padStart(2, '0');
  const day = `${date.getDate()}`.padStart(2, '0');
  return `${date.getFullYear()}-${month}-${day}`;
}

/* ------------------------------------------------------------------ misc */

export function initials(name: string | undefined | null, fallback = '?'): string {
  const parts = (name ?? '').trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return fallback;
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

export function compactNumber(n: number): string {
  return new Intl.NumberFormat('en', { notation: 'compact', maximumFractionDigits: 1 }).format(n);
}

/**
 * `time_limit` is a bare float with no documented unit: the admin update DTO
 * caps it at 30 (seconds), while seeded problems store 1000 (milliseconds).
 * Values of 50 or more are read as milliseconds, anything smaller as seconds.
 */
export function formatTimeLimit(value: number): string {
  if (!value) return '—';
  const ms = value >= 50 ? value : value * 1000;
  return ms >= 1000 ? `${+(ms / 1000).toFixed(2)} s` : `${Math.round(ms)} ms`;
}

export function formatMemoryLimit(mb: number): string {
  return mb ? `${mb} MB` : '—';
}

export function formatRuntimeMs(ms: number | null | undefined): string {
  if (ms === null || ms === undefined) return '—';
  return `${ms} ms`;
}

export function formatMemoryKb(kb: number | null | undefined): string {
  if (kb === null || kb === undefined) return '—';
  if (kb >= 1024) return `${(kb / 1024).toFixed(kb >= 10240 ? 1 : 2)} MB`;
  return `${kb} KB`;
}

export function formatTestcaseCount(
  passed: number | null | undefined,
  total: number | null | undefined,
): string {
  if (passed === null || passed === undefined || total === null || total === undefined) return '—';
  return `${passed}/${total}`;
}

export function greeting(date = new Date()): string {
  const hours = date.getHours();
  if (hours < 12) return 'Good morning';
  if (hours < 18) return 'Good afternoon';
  return 'Good evening';
}

/** Resolve an avatar URL that may be returned as a bare object key. */
export function avatarUrl(raw: string | null | undefined, base: string): string | null {
  if (!raw) return null;
  if (/^https?:\/\//i.test(raw)) return raw;
  return `${base}${raw.startsWith('/') ? '' : '/'}${raw}`;
}
