"use client";

/**
 * useFetch — Generic data fetching hook with FetchState discriminated union (plan.md §8, §17).
 * Stale-while-revalidate pattern. Refetch on window focus.
 */

import { useState, useEffect, useCallback, useRef } from "react";
import type { FetchState } from "@/types/state";

interface UseFetchOptions<T> {
  enabled?: boolean;
  onSuccess?: (data: T) => void;
  onError?: (err: Error) => void;
}

export function useFetch<T>(
  fetcher: () => Promise<T>,
  deps: React.DependencyList,
  options?: UseFetchOptions<T>
) {
  const { enabled = true, onSuccess, onError } = options || {};
  const [state, setState] = useState<FetchState<T>>({ status: "IDLE" });

  const mounted = useRef(true);
  const fetcherRef = useRef(fetcher);

  useEffect(() => {
    fetcherRef.current = fetcher;
  }, [fetcher]);

  const execute = useCallback(
    async (silent = false) => {
      if (!silent) {
        setState((prev) =>
          prev.status === "SUCCESS" ? prev : { status: "LOADING" }
        );
      }

      try {
        const data = await fetcherRef.current();
        if (!mounted.current) return;

        setState({ status: "SUCCESS", data, fetchedAt: Date.now() });
        onSuccess?.(data);
      } catch (err) {
        if (!mounted.current) return;

        const errorMsg =
          err instanceof Error ? err.message : "An error occurred";
        setState({ status: "ERROR", error: errorMsg, fetchedAt: Date.now() });
        onError?.(err instanceof Error ? err : new Error(errorMsg));
      }
    },
    [onSuccess, onError]
  );

  useEffect(() => {
    mounted.current = true;
    if (enabled) {
      execute();
    }
    return () => {
      mounted.current = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, enabled, execute]);

  return {
    status: state.status,
    data: "data" in state ? state.data : undefined,
    error: "error" in state ? state.error : undefined,
    fetchedAt: "fetchedAt" in state ? state.fetchedAt : undefined,
    refetch: () => execute(false),
    mutate: (data: T) =>
      setState({ status: "SUCCESS", data, fetchedAt: Date.now() }),
  };
}
