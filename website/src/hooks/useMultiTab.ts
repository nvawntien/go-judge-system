"use client";

/**
 * useMultiTab — Cross-tab state synchronization (plan.md §22).
 * Uses BroadcastChannel API. Falls back to localStorage events.
 */

import { useEffect, useRef, useCallback } from "react";

interface MultiTabEvent<T = unknown> {
  channel: string;
  type: string;
  payload: T;
  timestamp: number;
  tabId: string;
}

const TAB_ID = typeof crypto !== "undefined" ? crypto.randomUUID() : "ssr";

export function useMultiTab<T = unknown>(
  channelName: string,
  onMessage: (event: MultiTabEvent<T>) => void
) {
  const onMessageRef = useRef(onMessage);
  useEffect(() => {
    onMessageRef.current = onMessage;
  });

  const channelRef = useRef<BroadcastChannel | null>(null);

  const broadcast = useCallback(
    (type: string, payload: T) => {
      const event: MultiTabEvent<T> = {
        channel: channelName,
        type,
        payload,
        timestamp: Date.now(),
        tabId: TAB_ID,
      };

      // Try BroadcastChannel first
      if (channelRef.current) {
        channelRef.current.postMessage(event);
      }

      // Also write to localStorage as fallback (for browsers without BroadcastChannel)
      try {
        localStorage.setItem(
          `oj-multitab-${channelName}`,
          JSON.stringify(event)
        );
      } catch {
        // localStorage may be unavailable
      }
    },
    [channelName]
  );

  useEffect(() => {
    // BroadcastChannel
    if (typeof BroadcastChannel !== "undefined") {
      const bc = new BroadcastChannel(channelName);
      channelRef.current = bc;

      bc.onmessage = (e: MessageEvent<MultiTabEvent<T>>) => {
        if (e.data.tabId !== TAB_ID) {
          onMessageRef.current(e.data);
        }
      };
    }

    // localStorage fallback
    const handleStorage = (e: StorageEvent) => {
      if (e.key === `oj-multitab-${channelName}` && e.newValue) {
        try {
          const event: MultiTabEvent<T> = JSON.parse(e.newValue);
          if (event.tabId !== TAB_ID) {
            onMessageRef.current(event);
          }
        } catch {
          // Ignore parse errors
        }
      }
    };

    window.addEventListener("storage", handleStorage);

    return () => {
      channelRef.current?.close();
      channelRef.current = null;
      window.removeEventListener("storage", handleStorage);
    };
  }, [channelName]);

  return { broadcast, tabId: TAB_ID };
}
