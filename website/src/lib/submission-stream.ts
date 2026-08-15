'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { buildApiUrl, submissionApi } from './api';
import { isTerminalSubmissionStatus } from './format';
import type { SubmissionStreamEvent, SubmissionStatus } from './types';

export type SubmissionStreamState =
  | 'idle'
  | 'requesting_ticket'
  | 'connecting'
  | 'open'
  | 'reconnecting'
  | 'completed'
  | 'error';

export interface SubmissionStreamError {
  kind: 'ticket' | 'stream' | 'malformed_event';
  message: string;
}

interface UseSubmissionStreamOptions {
  submissionId: number | null;
  initialAttemptId?: string | null;
  enabled: boolean;
  onStatus: (event: SubmissionStreamEvent) => void;
  onTerminal: (event: SubmissionStreamEvent) => void;
  onError?: (error: SubmissionStreamError) => void;
}

interface UseSubmissionStreamResult {
  state: SubmissionStreamState;
  error: SubmissionStreamError | null;
  reconnect: () => void;
}

const MAX_RECONNECT_ATTEMPTS = 5;
const BASE_RECONNECT_DELAY_MS = 1000;
const MAX_RECONNECT_DELAY_MS = 10_000;

const SUBMISSION_STREAM_EVENTS = [
  'submission.snapshot',
  'submission.updated',
  'submission.completed',
] as const;

function buildSubmissionStreamURL(submissionId: number, ticket: string): string {
  return buildApiUrl(`/events/submissions/${submissionId}`, { ticket });
}

function isSubmissionStreamEvent(value: unknown): value is SubmissionStreamEvent {
  if (!value || typeof value !== 'object') return false;
  const item = value as Partial<SubmissionStreamEvent>;
  return (
    typeof item.submission_id === 'number' &&
    typeof item.attempt_id === 'string' &&
    typeof item.status === 'string' &&
    isKnownSubmissionStatus(item.status) &&
    typeof item.updated_at === 'string'
  );
}

function isKnownSubmissionStatus(value: string): value is SubmissionStatus {
  return (
    value === 'PENDING' ||
    value === 'JUDGING' ||
    value === 'ACCEPTED' ||
    value === 'WRONG_ANSWER' ||
    value === 'TIME_LIMIT_EXCEEDED' ||
    value === 'MEMORY_LIMIT_EXCEEDED' ||
    value === 'OUTPUT_LIMIT_EXCEEDED' ||
    value === 'RUNTIME_ERROR' ||
    value === 'COMPILATION_ERROR' ||
    value === 'SYSTEM_ERROR'
  );
}

function reconnectDelay(attempt: number): number {
  const base = Math.min(MAX_RECONNECT_DELAY_MS, BASE_RECONNECT_DELAY_MS * 2 ** Math.max(0, attempt - 1));
  const jitter = Math.floor(Math.random() * 250);
  return base + jitter;
}

export function useSubmissionStream({
  submissionId,
  initialAttemptId = null,
  enabled,
  onStatus,
  onTerminal,
  onError,
}: UseSubmissionStreamOptions): UseSubmissionStreamResult {
  const [state, setState] = useState<SubmissionStreamState>('idle');
  const [error, setError] = useState<SubmissionStreamError | null>(null);
  const [reconnectNonce, setReconnectNonce] = useState(0);

  const eventSourceRef = useRef<EventSource | null>(null);
  const ticketAbortRef = useRef<AbortController | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const generationRef = useRef(0);
  const terminalHandledRef = useRef(false);
  const activeAttemptRef = useRef<string | null>(initialAttemptId ?? null);
  const onStatusRef = useRef(onStatus);
  const onTerminalRef = useRef(onTerminal);
  const onErrorRef = useRef(onError);

  useEffect(() => {
    onStatusRef.current = onStatus;
  }, [onStatus]);

  useEffect(() => {
    onTerminalRef.current = onTerminal;
  }, [onTerminal]);

  useEffect(() => {
    onErrorRef.current = onError;
  }, [onError]);

  const reconnect = useCallback(() => {
    setReconnectNonce((current) => current + 1);
  }, []);

  useEffect(() => {
    generationRef.current += 1;
    const generation = generationRef.current;
    terminalHandledRef.current = false;
    activeAttemptRef.current = initialAttemptId ?? null;
    setError(null);

    const clearReconnectTimer = () => {
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
        reconnectTimerRef.current = null;
      }
    };

    const closeEventSource = () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
    };

    const abortTicketRequest = () => {
      if (ticketAbortRef.current) {
        ticketAbortRef.current.abort();
        ticketAbortRef.current = null;
      }
    };

    const reportError = (next: SubmissionStreamError) => {
      setError(next);
      onErrorRef.current?.(next);
    };

    const validateEvent = (event: SubmissionStreamEvent): boolean => {
      if (event.submission_id !== submissionId) return false;
      if (!activeAttemptRef.current) {
        activeAttemptRef.current = event.attempt_id;
        return true;
      }
      if (event.attempt_id !== activeAttemptRef.current) {
        if (process.env.NODE_ENV !== 'production') {
          console.warn('ignored stale submission stream event', {
            submission_id: event.submission_id,
            status: event.status,
          });
        }
        return false;
      }
      return true;
    };

    const complete = (event: SubmissionStreamEvent) => {
      if (terminalHandledRef.current) return;
      terminalHandledRef.current = true;
      closeEventSource();
      clearReconnectTimer();
      setState('completed');
      onTerminalRef.current(event);
    };

    const handleRawEvent = (message: MessageEvent<string>) => {
      if (generationRef.current !== generation) return;
      let parsed: unknown;
      try {
        parsed = JSON.parse(message.data) as unknown;
      } catch {
        closeEventSource();
        setState('error');
        reportError({ kind: 'malformed_event', message: 'Live update payload was not valid JSON.' });
        return;
      }

      if (!isSubmissionStreamEvent(parsed)) {
        closeEventSource();
        setState('error');
        reportError({ kind: 'malformed_event', message: 'Live update payload did not match the contract.' });
        return;
      }

      if (!validateEvent(parsed)) return;
      onStatusRef.current(parsed);
      if (isTerminalSubmissionStatus(parsed.status)) complete(parsed);
    };

    const connect = async (attempt: number) => {
      if (!enabled || !submissionId || generationRef.current !== generation) return;

      abortTicketRequest();
      closeEventSource();
      clearReconnectTimer();
      setState(attempt === 0 ? 'requesting_ticket' : 'reconnecting');
      setError(null);

      const controller = new AbortController();
      ticketAbortRef.current = controller;

      let ticket: string;
      try {
        const response = await submissionApi.issueStreamTicket(submissionId, controller.signal);
        ticket = response.ticket;
      } catch (err) {
        if (controller.signal.aborted || generationRef.current !== generation) return;
        if (attempt < MAX_RECONNECT_ATTEMPTS) {
          reconnectTimerRef.current = setTimeout(() => {
            void connect(attempt + 1);
          }, reconnectDelay(attempt + 1));
          return;
        }
        setState('error');
        reportError({ kind: 'ticket', message: 'Could not request a live update ticket.' });
        return;
      } finally {
        if (ticketAbortRef.current === controller) ticketAbortRef.current = null;
      }

      if (generationRef.current !== generation || terminalHandledRef.current) return;

      setState('connecting');
      const source = new EventSource(buildSubmissionStreamURL(submissionId, ticket));
      eventSourceRef.current = source;

      source.onopen = () => {
        if (generationRef.current !== generation || terminalHandledRef.current) return;
        setState('open');
      };

      source.onerror = () => {
        if (generationRef.current !== generation || terminalHandledRef.current) {
          source.close();
          return;
        }
        source.close();
        if (eventSourceRef.current === source) eventSourceRef.current = null;

        if (attempt < MAX_RECONNECT_ATTEMPTS) {
          setState('reconnecting');
          reconnectTimerRef.current = setTimeout(() => {
            void connect(attempt + 1);
          }, reconnectDelay(attempt + 1));
          return;
        }

        setState('error');
        reportError({ kind: 'stream', message: 'Live updates are unavailable.' });
      };

      for (const eventName of SUBMISSION_STREAM_EVENTS) {
        source.addEventListener(eventName, handleRawEvent as EventListener);
      }
    };

    if (enabled && submissionId) {
      void connect(0);
    } else {
      setState('idle');
    }

    return () => {
      generationRef.current += 1;
      abortTicketRequest();
      closeEventSource();
      clearReconnectTimer();
    };
  }, [enabled, initialAttemptId, reconnectNonce, submissionId]);

  return { state, error, reconnect };
}
