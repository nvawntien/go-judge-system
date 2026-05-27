/**
 * Optimistic update helpers (plan.md §19).
 *
 * Usage:
 *   const { execute } = useOptimistic(cache);
 *   execute(
 *     () => api.updateSomething(data),     // mutation
 *     (prev) => ({ ...prev, ...data }),     // optimistic updater
 *     { onError: () => addToast("error", "Failed") }
 *   );
 */

import { useCallback, useRef } from "react";

interface OptimisticOptions<T> {
  /** Called after successful mutation */
  onSuccess?: (result: T) => void;
  /** Called on error (after rollback) */
  onError?: (error: Error) => void;
}

/**
 * useOptimistic — Wraps a cache's mutate function with optimistic update + rollback.
 *
 * @param mutate   - The cache mutate function (from useCache)
 * @param getData  - Returns current data snapshot for rollback
 */
export function useOptimistic<T>(
  mutate: (updater: T | ((prev: T | undefined) => T)) => void,
  getData: () => T | undefined
) {
  const snapshotRef = useRef<T | undefined>(undefined);

  const execute = useCallback(
    async <R>(
      mutation: () => Promise<R>,
      optimisticUpdater: (prev: T | undefined) => T,
      options?: OptimisticOptions<R>
    ): Promise<R | undefined> => {
      // 1. Snapshot current state for rollback
      snapshotRef.current = getData();

      // 2. Apply optimistic update immediately
      mutate(optimisticUpdater);

      try {
        // 3. Execute real mutation
        const result = await mutation();
        options?.onSuccess?.(result);
        return result;
      } catch (err) {
        // 4. Rollback on error
        if (snapshotRef.current !== undefined) {
          mutate(snapshotRef.current);
        }
        const error = err instanceof Error ? err : new Error("Mutation failed");
        options?.onError?.(error);
        return undefined;
      }
    },
    [mutate, getData]
  );

  return { execute };
}
