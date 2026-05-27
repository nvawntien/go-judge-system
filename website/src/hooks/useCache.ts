"use client";

/**
 * useCache — Server state cache with stale-while-revalidate (plan.md §17).
 * Deterministic cache keys. Refetch on window focus. Invalidate on mutation.
 * Separates UI state from server state.
 */

import { useCallback, useEffect, useRef, useState } from "react";

interface CacheEntry<T> {
  data: T;
  fetchedAt: number;
  staleAfter: number;
}

// Global in-memory cache store (shared across components)
const globalCache = new Map<string, CacheEntry<unknown>>();
const subscribers = new Map<string, Set<() => void>>();

function notifySubscribers(key: string) {
  const subs = subscribers.get(key);
  if (subs) {
    subs.forEach((cb) => cb());
  }
}

/** Invalidate a cache key (e.g. after mutation) */
export function invalidateCache(key: string) {
  globalCache.delete(key);
  notifySubscribers(key);
}

/** Invalidate all cache keys matching a prefix */
export function invalidateCachePrefix(prefix: string) {
  for (const key of globalCache.keys()) {
    if (key.startsWith(prefix)) {
      globalCache.delete(key);
    }
  }
  // Notify all subscribers since we don't know which ones changed
  for (const [key, subs] of subscribers.entries()) {
    if (key.startsWith(prefix)) {
      subs.forEach((cb) => cb());
    }
  }
}

interface UseCacheOptions {
  /** Stale time in ms (default: 30000 = 30s) */
  staleTime?: number;
  /** Whether to refetch on window focus (default: true) */
  refetchOnFocus?: boolean;
  /** Whether cache is enabled (default: true) */
  enabled?: boolean;
}

export function useCache<T>(
  cacheKey: string[],
  fetcher: () => Promise<T>,
  options?: UseCacheOptions
) {
  const {
    staleTime = 30000,
    refetchOnFocus = true,
    enabled = true,
  } = options || {};

  const key = cacheKey.join("::");
  const fetcherRef = useRef(fetcher);
  useEffect(() => {
    fetcherRef.current = fetcher;
  }, [fetcher]);

  const [data, setData] = useState<T | undefined>(() => {
    const cached = globalCache.get(key) as CacheEntry<T> | undefined;
    return cached?.data;
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | undefined>();
  const mountedRef = useRef(true);

  const fetchData = useCallback(
    async (force = false) => {
      if (!enabled) return;

      // Check cache first
      if (!force) {
        const cached = globalCache.get(key) as CacheEntry<T> | undefined;
        if (cached && Date.now() < cached.staleAfter) {
          setData(cached.data);
          return;
        }
        // Stale but exists — show stale data while revalidating
        if (cached) {
          setData(cached.data);
        }
      }

      setIsLoading(true);
      try {
        const result = await fetcherRef.current();
        if (!mountedRef.current) return;

        const entry: CacheEntry<T> = {
          data: result,
          fetchedAt: Date.now(),
          staleAfter: Date.now() + staleTime,
        };
        globalCache.set(key, entry as CacheEntry<unknown>);
        setData(result);
        setError(undefined);
      } catch (err) {
        if (!mountedRef.current) return;
        setError(err instanceof Error ? err.message : "Fetch failed");
      } finally {
        if (mountedRef.current) {
          setIsLoading(false);
        }
      }
    },
    [key, staleTime, enabled]
  );

  // Subscribe to invalidation events
  useEffect(() => {
    mountedRef.current = true;

    if (!subscribers.has(key)) {
      subscribers.set(key, new Set());
    }
    const cb = () => fetchData(true);
    subscribers.get(key)!.add(cb);

    return () => {
      mountedRef.current = false;
      subscribers.get(key)?.delete(cb);
      if (subscribers.get(key)?.size === 0) {
        subscribers.delete(key);
      }
    };
  }, [key, fetchData]);

  // Initial fetch
  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // Refetch on window focus
  useEffect(() => {
    if (!refetchOnFocus) return;

    const handleFocus = () => {
      // Only refetch if data is stale
      const cached = globalCache.get(key) as CacheEntry<T> | undefined;
      if (!cached || Date.now() >= cached.staleAfter) {
        fetchData();
      }
    };

    window.addEventListener("focus", handleFocus);
    return () => window.removeEventListener("focus", handleFocus);
  }, [key, refetchOnFocus, fetchData]);

  // Refetch on network reconnect
  useEffect(() => {
    const handleOnline = () => fetchData(true);
    window.addEventListener("online", handleOnline);
    return () => window.removeEventListener("online", handleOnline);
  }, [fetchData]);

  return {
    data,
    isLoading,
    error,
    refetch: () => fetchData(true),
    /** Optimistically update cache data */
    mutate: (updater: T | ((prev: T | undefined) => T)) => {
      const newData =
        typeof updater === "function"
          ? (updater as (prev: T | undefined) => T)(data)
          : updater;
      setData(newData);
      globalCache.set(key, {
        data: newData,
        fetchedAt: Date.now(),
        staleAfter: Date.now() + staleTime,
      } as CacheEntry<unknown>);
    },
  };
}
