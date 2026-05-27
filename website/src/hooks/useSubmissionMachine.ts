"use client";

/**
 * useSubmissionMachine — Discriminated union state machine (plan.md §15).
 * States: IDLE → SUBMITTING → QUEUED → RUNNING → RESULT | ERROR
 * Uses useReducer with exhaustive switch.
 */

import { useReducer, useCallback } from "react";
import type { SubmissionMachineState, SubmissionAction } from "@/types/state";
import { submissionApi } from "@/lib/api-client";
import type { CreateSubmissionRequest } from "@/types/api";

function submissionReducer(
  state: SubmissionMachineState,
  action: SubmissionAction
): SubmissionMachineState {
  switch (action.type) {
    case "SUBMIT":
      return { status: "SUBMITTING", idempotencyKey: action.idempotencyKey };

    case "QUEUED":
      if (state.status !== "SUBMITTING") return state; // invalid transition
      return {
        status: "QUEUED",
        submissionId: action.submissionId,
        queuedAt: new Date().toISOString(),
      };

    case "RUNNING":
      if (state.status !== "QUEUED" && state.status !== "SUBMITTING")
        return state;
      return {
        status: "RUNNING",
        submissionId: action.submissionId,
        startedAt: new Date().toISOString(),
      };

    case "RESULT":
      return {
        status: "RESULT",
        verdict: action.detail.status,
        detail: action.detail,
      };

    case "ERROR":
      return { status: "ERROR", error: action.error };

    case "RESET":
      return { status: "IDLE" };

    default: {
      const _exhaustive: never = action;
      return _exhaustive;
    }
  }
}

export function useSubmissionMachine() {
  const [state, dispatch] = useReducer(submissionReducer, { status: "IDLE" });

  const pollResult = useCallback((submissionId: number) => {
    let attempts = 0;
    const maxAttempts = 30;

    const interval = setInterval(async () => {
      attempts++;
      try {
        const detail = await submissionApi.getMyDetail(submissionId);

        if (detail.status === "JUDGING" || detail.status === "PENDING") {
          if (detail.status === "JUDGING") {
            dispatch({ type: "RUNNING", submissionId });
          }
          if (attempts >= maxAttempts) {
            clearInterval(interval);
            dispatch({
              type: "ERROR",
              error: "Polling timeout. Check your submissions later.",
            });
          }
          return;
        }

        clearInterval(interval);
        dispatch({ type: "RESULT", detail });
      } catch (err) {
        clearInterval(interval);
        dispatch({
          type: "ERROR",
          error:
            err instanceof Error
              ? err.message
              : "Failed to fetch submission result",
        });
      }
    }, 1000);
  }, []);

  const submit = useCallback(
    async (data: CreateSubmissionRequest) => {
      const idempotencyKey = crypto.randomUUID();
      dispatch({ type: "SUBMIT", idempotencyKey });

      try {
        const res = await submissionApi.create(data, idempotencyKey);
        dispatch({ type: "QUEUED", submissionId: res.id });
        pollResult(res.id);
      } catch (err) {
        dispatch({
          type: "ERROR",
          error: err instanceof Error ? err.message : "Submission failed",
        });
      }
    },
    [pollResult]
  );

  const reset = useCallback(() => {
    dispatch({ type: "RESET" });
  }, []);

  return { state, submit, reset };
}
