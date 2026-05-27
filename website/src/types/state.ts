/**
 * State Machine types — Discriminated unions (plan.md §15).
 * Each state carries only relevant data. No scattered booleans.
 */

import type { SubmissionDetailResponse, SubmissionStatus } from "./api";

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//  GENERIC FETCH STATE MACHINE
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

export type FetchState<T> =
  | { status: "IDLE" }
  | { status: "LOADING" }
  | { status: "SUCCESS"; data: T; fetchedAt: number }
  | { status: "ERROR"; error: string; fetchedAt: number };

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//  AUTH STATE MACHINE
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

export interface AuthUser {
  userId: string;
  username: string;
  role: string;
}

export type AuthState =
  | { status: "IDLE" }
  | { status: "AUTHENTICATING" }
  | { status: "AUTHENTICATED"; user: AuthUser }
  | { status: "ERROR"; error: string };

export type AuthAction =
  | { type: "LOGIN_START" }
  | { type: "LOGIN_SUCCESS"; user: AuthUser }
  | { type: "LOGIN_ERROR"; error: string }
  | { type: "LOGOUT" }
  | { type: "RESTORE"; user: AuthUser };

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//  SUBMISSION STATE MACHINE (plan.md §15)
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

export type SubmissionMachineState =
  | { status: "IDLE" }
  | { status: "SUBMITTING"; idempotencyKey: string }
  | { status: "QUEUED"; submissionId: number; queuedAt: string }
  | { status: "RUNNING"; submissionId: number; startedAt: string }
  | {
      status: "RESULT";
      verdict: SubmissionStatus;
      detail: SubmissionDetailResponse;
    }
  | { status: "ERROR"; error: string };

export type SubmissionAction =
  | { type: "SUBMIT"; idempotencyKey: string }
  | { type: "QUEUED"; submissionId: number }
  | { type: "RUNNING"; submissionId: number }
  | { type: "RESULT"; detail: SubmissionDetailResponse }
  | { type: "ERROR"; error: string }
  | { type: "RESET" };

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//  NETWORK STATUS (plan.md §24)
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

export type NetworkStatus = "ONLINE" | "DEGRADED" | "OFFLINE";
