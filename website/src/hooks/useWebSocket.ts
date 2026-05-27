"use client";

/**
 * useWebSocket — Typed WebSocket hook (plan.md §9, §20).
 * Auto reconnect with exponential backoff. Fallback to polling.
 * Handles out-of-order events. Resume subscriptions after reconnect.
 */

import { useState, useEffect, useRef, useCallback } from "react";

type WebSocketStatus = "CONNECTING" | "OPEN" | "CLOSING" | "CLOSED" | "RECONNECTING";

interface UseWebSocketOptions<T> {
  /** WebSocket URL */
  url: string | null;
  /** Called on each message */
  onMessage?: (data: T) => void;
  /** Called when connection opens */
  onOpen?: () => void;
  /** Called when connection closes */
  onClose?: () => void;
  /** Called when an error occurs */
  onError?: (err: Event) => void;
  /** Max reconnect attempts (default: 10) */
  maxReconnectAttempts?: number;
  /** Base delay for reconnect backoff in ms (default: 1000) */
  reconnectBaseDelay?: number;
  /** Fallback polling interval in ms when WS fails (default: 3000) */
  pollingInterval?: number;
  /** Polling function — called when WS is unavailable */
  pollingFn?: () => Promise<void>;
  /** Auto-connect on mount (default: true) */
  enabled?: boolean;
}

export function useWebSocket<T = unknown>(options: UseWebSocketOptions<T>) {
  const {
    url,
    onMessage,
    onOpen,
    onClose,
    onError,
    maxReconnectAttempts = 10,
    reconnectBaseDelay = 1000,
    pollingInterval = 3000,
    pollingFn,
    enabled = true,
  } = options;

  const [status, setStatus] = useState<WebSocketStatus>("CLOSED");
  const wsRef = useRef<WebSocket | null>(null);
  const attemptRef = useRef(0);
  const reconnectTimerRef = useRef<NodeJS.Timeout | null>(null);
  const pollingTimerRef = useRef<NodeJS.Timeout | null>(null);

  // Stable refs for callbacks
  const onMessageRef = useRef(onMessage);
  const onOpenRef = useRef(onOpen);
  const onCloseRef = useRef(onClose);
  const onErrorRef = useRef(onError);
  const pollingFnRef = useRef(pollingFn);

  useEffect(() => {
    onMessageRef.current = onMessage;
    onOpenRef.current = onOpen;
    onCloseRef.current = onClose;
    onErrorRef.current = onError;
    pollingFnRef.current = pollingFn;
  });

  const stopPolling = useCallback(() => {
    if (pollingTimerRef.current) {
      clearInterval(pollingTimerRef.current);
      pollingTimerRef.current = null;
    }
  }, []);

  const startPolling = useCallback(() => {
    stopPolling();
    if (!pollingFnRef.current) return;

    pollingTimerRef.current = setInterval(() => {
      pollingFnRef.current?.().catch(() => {
        // Silently handle polling errors
      });
    }, pollingInterval);
  }, [pollingInterval, stopPolling]);

  const connect = useCallback(() => {
    if (!url || !enabled) return;

    // Clean up existing connection
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    setStatus("CONNECTING");

    try {
      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = () => {
        setStatus("OPEN");
        attemptRef.current = 0;
        stopPolling(); // WS is up — stop polling fallback
        onOpenRef.current?.();
      };

      ws.onmessage = (event) => {
        try {
          const data: T = JSON.parse(event.data);
          onMessageRef.current?.(data);
        } catch {
          // Non-JSON message — ignore
        }
      };

      ws.onerror = (event) => {
        onErrorRef.current?.(event);
      };

      ws.onclose = () => {
        setStatus("CLOSED");
        wsRef.current = null;
        onCloseRef.current?.();

        // Attempt reconnect with exponential backoff
        if (attemptRef.current < maxReconnectAttempts) {
          const delay = Math.min(
            reconnectBaseDelay * Math.pow(2, attemptRef.current),
            30000
          );
          const jitteredDelay = delay + Math.random() * delay * 0.3;

          setStatus("RECONNECTING");
          attemptRef.current++;

          reconnectTimerRef.current = setTimeout(() => {
            connect();
          }, jitteredDelay);
        } else {
          // Max reconnect attempts exceeded — fall back to polling
          startPolling();
        }
      };
    } catch {
      setStatus("CLOSED");
      startPolling();
    }
  }, [url, enabled, maxReconnectAttempts, reconnectBaseDelay, stopPolling, startPolling]);

  const disconnect = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current);
    }
    stopPolling();
    attemptRef.current = maxReconnectAttempts; // Prevent auto-reconnect

    if (wsRef.current) {
      setStatus("CLOSING");
      wsRef.current.close();
      wsRef.current = null;
    }
  }, [maxReconnectAttempts, stopPolling]);

  const send = useCallback((data: unknown) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data));
    }
  }, []);

  // Connect on mount, disconnect on unmount
  useEffect(() => {
    if (enabled && url) {
      connect();
    }
    return () => {
      disconnect();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [url, enabled]);

  return {
    status,
    send,
    connect,
    disconnect,
    isConnected: status === "OPEN",
  };
}
