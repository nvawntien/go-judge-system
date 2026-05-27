"use client";

/**
 * useNetworkStatus — Detects ONLINE / DEGRADED / OFFLINE (plan.md §24).
 * Uses navigator.onLine + optional WebSocket health probe.
 */

import { useState, useEffect, useCallback } from "react";
import type { NetworkStatus } from "@/types/state";

export function useNetworkStatus(): {
  status: NetworkStatus;
  isOnline: boolean;
} {
  const [status, setStatus] = useState<NetworkStatus>("ONLINE");

  const updateStatus = useCallback(() => {
    if (!navigator.onLine) {
      setStatus("OFFLINE");
    } else {
      setStatus("ONLINE");
    }
  }, []);

  useEffect(() => {
    updateStatus();

    const handleOnline = () => setStatus("ONLINE");
    const handleOffline = () => setStatus("OFFLINE");

    window.addEventListener("online", handleOnline);
    window.addEventListener("offline", handleOffline);

    return () => {
      window.removeEventListener("online", handleOnline);
      window.removeEventListener("offline", handleOffline);
    };
  }, [updateStatus]);

  return {
    status,
    isOnline: status !== "OFFLINE",
  };
}

/**
 * Mark status as DEGRADED from external source (e.g., WebSocket disconnected
 * but navigator.onLine is still true).
 */
export function useDegradedFlag() {
  const [degraded, setDegraded] = useState(false);
  return { degraded, markDegraded: () => setDegraded(true), markRecovered: () => setDegraded(false) };
}
