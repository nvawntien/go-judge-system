'use client';

import { useCallback, useEffect, useState } from 'react';
import { ApiError, submissionApi } from './api';
import type { SubmissionListItem } from './types';

/**
 * Detailed per-problem progress is derived from the caller's own submission
 * history for routes that need solved/attempted problem IDs. Profile aggregate
 * statistics use Submission's dedicated /me/profile-stats endpoint instead.
 * Results are memoised per page-session; `invalidateProgress()` is called
 * after a new submission lands.
 */

const MAX_PAGES = 10;
const PAGE_SIZE = 100;
const TTL_MS = 60_000;

export interface Progress {
  submissions: SubmissionListItem[];
  solvedIds: Set<number>;
  attemptedIds: Set<number>;
  /** True when the history was truncated at MAX_PAGES * PAGE_SIZE. */
  truncated: boolean;
}

let progressCache: { at: number; value: Promise<Progress> } | null = null;

async function loadProgress(): Promise<Progress> {
  const submissions: SubmissionListItem[] = [];
  let page = 1;
  let truncated = false;

  for (; page <= MAX_PAGES; page += 1) {
    const res = await submissionApi.listMine({ page, limit: PAGE_SIZE });
    submissions.push(...res.items);
    if (page >= (res.pagination?.total_pages ?? 1)) break;
    if (page === MAX_PAGES) truncated = true;
  }

  const solvedIds = new Set<number>();
  const attemptedIds = new Set<number>();
  for (const submission of submissions) {
    attemptedIds.add(submission.problem_id);
    if (submission.status === 'ACCEPTED') solvedIds.add(submission.problem_id);
  }

  return { submissions, solvedIds, attemptedIds, truncated };
}

export function fetchProgress(force = false): Promise<Progress> {
  const fresh = progressCache && Date.now() - progressCache.at < TTL_MS;
  if (!force && fresh && progressCache) return progressCache.value;

  const value = loadProgress().catch((err) => {
    progressCache = null;
    throw err;
  });
  progressCache = { at: Date.now(), value };
  return value;
}

export function invalidateProgress() {
  progressCache = null;
}

const EMPTY: Progress = {
  submissions: [],
  solvedIds: new Set(),
  attemptedIds: new Set(),
  truncated: false,
};

/** Loads the signed-in user's progress; resolves to empty sets when signed out. */
export function useProgress(enabled: boolean): {
  progress: Progress;
  loading: boolean;
  reload: () => void;
} {
  const [progress, setProgress] = useState<Progress>(EMPTY);
  const [loading, setLoading] = useState(enabled);
  const [nonce, setNonce] = useState(0);

  useEffect(() => {
    if (!enabled) {
      setProgress(EMPTY);
      setLoading(false);
      return;
    }

    let active = true;
    setLoading(true);
    fetchProgress(nonce > 0)
      .then((value) => {
        if (active) setProgress(value);
      })
      .catch((err) => {
        if (!(err instanceof ApiError && err.isUnauthorized)) {
          // Progress is decorative — a failure must not break the page.
          console.warn('progress load failed', err);
        }
        if (active) setProgress(EMPTY);
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [enabled, nonce]);

  const reload = useCallback(() => {
    invalidateProgress();
    setNonce((n) => n + 1);
  }, []);

  return { progress, loading, reload };
}
