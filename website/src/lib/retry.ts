/**
 * Retry with exponential backoff (plan.md §25).
 * Max retry: 5. Jitter applied. NO retry on 4xx errors.
 */

import { ApiError } from "@/lib/api-client";

interface RetryOptions {
  maxAttempts?: number;
  baseDelayMs?: number;
  maxDelayMs?: number;
}

const DEFAULT_OPTIONS: Required<RetryOptions> = {
  maxAttempts: 5,
  baseDelayMs: 500,
  maxDelayMs: 15000,
};

function jitter(delay: number): number {
  return delay + Math.random() * delay * 0.3;
}

function isRetryable(err: unknown): boolean {
  // Do NOT retry on 4xx client errors
  if (err instanceof ApiError && err.statusCode >= 400 && err.statusCode < 500) {
    return false;
  }
  return true;
}

export async function withRetry<T>(
  fn: () => Promise<T>,
  opts?: RetryOptions
): Promise<T> {
  const { maxAttempts, baseDelayMs, maxDelayMs } = {
    ...DEFAULT_OPTIONS,
    ...opts,
  };

  let lastError: unknown;

  for (let attempt = 0; attempt < maxAttempts; attempt++) {
    try {
      return await fn();
    } catch (err) {
      lastError = err;

      if (!isRetryable(err) || attempt === maxAttempts - 1) {
        throw err;
      }

      const delay = Math.min(baseDelayMs * Math.pow(2, attempt), maxDelayMs);
      await new Promise((resolve) => setTimeout(resolve, jitter(delay)));
    }
  }

  throw lastError;
}
