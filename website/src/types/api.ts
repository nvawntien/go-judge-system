/**
 * TypeScript interfaces — exact match to Go backend DTOs.
 *
 * Sources:
 *   Auth:       services/auth/internal/application/dto/
 *   Problem:    services/problem/internal/application/dto/
 *   Submission: services/submission/internal/application/dto/
 *   Response:   pkg/response/response.go
 *
 * ZERO-GUESS POLICY: field names mirror JSON tags from Go structs.
 */

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//  SHARED — API Response wrapper
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

export interface APIResponse<T = unknown> {
  status: "success" | "error";
  code: number;
  msg: string;
  data?: T;
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//  SHARED — Paginated list wrapper
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

export interface PaginatedList<T> {
  items: T[];
  total: number;
  page: number;
  limit: number;
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//  ENUMS — exact backend values
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

/** entity.Language — services/submission/internal/domain/entity/submission.go */
export type Language = "C" | "CPP" | "JAVA" | "PYTHON" | "GO" | "JAVASCRIPT";

/** entity.Status — services/submission/internal/domain/entity/submission.go */
export type SubmissionStatus =
  | "PENDING"
  | "JUDGING"
  | "ACCEPTED"
  | "WRONG_ANSWER"
  | "TIME_LIMIT_EXCEEDED"
  | "MEMORY_LIMIT_EXCEEDED"
  | "RUNTIME_ERROR"
  | "COMPILATION_ERROR"
  | "SYSTEM_ERROR";

/** dto.CreateProblemRequest.Difficulty — services/problem/internal/application/dto/problem.go */
export type Difficulty = "EASY" | "MEDIUM" | "HARD";

/** auth.Claims.Role — pkg/auth/claims.go */
export type Role = "user" | "admin" | "super_admin";

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//  AUTH DTOs — services/auth/internal/application/dto/
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

/** dto.LoginRequest — dto/login.go */
export interface LoginRequest {
  identifier: string; // can be email or username
  password: string;
}

/** dto.LoginResponse — dto/login.go */
export interface LoginResponse {
  access_token: string;
  access_expire: number;
  refresh_token: string;
  refresh_expire: number;
}

/** dto.RegisterRequest — dto/register.go */


export interface RegisterRequest {
  full_name: string;
  username: string;
  email: string;
  password: string;
}

/** dto.ProfileResponse — dto/profile.go */
export interface ProfileResponse {
  username: string;
  email: string;
  rating: number;
  created_at: string;
}

/** dto.VerifyEmailRequest — dto/verify_email.go */
export interface VerifyEmailRequest {
  token: string;
}

/** dto.ResendVerificationRequest — dto/resend_verification.go */
export interface ResendVerificationRequest {
  email: string;
}

/** dto.ForgotPasswordRequest — dto/forgot_password.go */
export interface ForgotPasswordRequest {
  email: string;
}

/** dto.ResetPasswordRequest — dto/reset_password.go */
export interface ResetPasswordRequest {
  token: string;
  new_password: string;
  confirm_password: string;
}

/** dto.ChangePasswordRequest — dto/change_password.go */
export interface ChangePasswordRequest {
  current_password: string;
  new_password: string;
  confirm_password: string;
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//  PROBLEM DTOs — services/problem/internal/application/dto/
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

/** dto.ProblemExampleDTO — dto/problem.go */
export interface ProblemExampleDTO {
  input: string;
  output: string;
  explanation?: string;
}

export interface CreateProblemRequest {
  title: string;
  slug: string;
  description: string;
  difficulty: Difficulty;
  examples: ProblemExampleDTO[];
  constraints?: string;
  hints?: string[];
  time_limit: number;
  memory_limit: number;
}

export interface CreateProblemResponse {
  id: number;
  slug: string;
}

export interface UpdateProblemRequest {
  title?: string;
  slug?: string;
  description?: string;
  difficulty?: Difficulty;
  examples?: ProblemExampleDTO[];
  constraints?: string;
  hints?: string[];
  time_limit?: number;
  memory_limit?: number;
}

/** dto.ProblemResponse — dto/problem.go */
export interface ProblemResponse {
  id: number;
  slug: string;
  title: string;
  description: string;
  difficulty: Difficulty;
  examples?: ProblemExampleDTO[];
  constraints?: string;
  hints?: string[];
  time_limit: number;
  memory_limit: number;
  author_id?: string;
  is_hidden?: boolean;
  created_at: string;
}

/** dto.ListProblemsResponse — dto/problem.go */
export type ListProblemsResponse = PaginatedList<ProblemResponse>;

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//  SUBMISSION DTOs — services/submission/internal/application/dto/
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

/** dto.CreateSubmissionRequest — dto/submission.go */
export interface CreateSubmissionRequest {
  problem_id: number;
  problem_name: string;
  language: Language;
  source_code: string;
}

/** dto.SubmissionResponse — dto/submission.go */
export interface SubmissionResponse {
  id: number;
  problem_id: number;
  problem_name: string;
  user_id: string;
  username: string;
  language: Language;
  status: SubmissionStatus;
  created_at: string;
}

/** dto.SubmissionResultResponse — dto/submission.go */
export interface SubmissionResultResponse {
  test_index: number;
  status: string;
  input?: string;
  expected_output?: string;
  actual_output?: string;
  execution_time_ms?: number;
  memory_used_kb?: number;
}

/** dto.SubmissionDetailResponse — dto/submission.go */
export interface SubmissionDetailResponse extends SubmissionResponse {
  source_code: string;
  execution_time_ms?: number;
  memory_used_kb?: number;
  compile_output?: string;
  total_tests: number;
  failed_test_index?: number;
  failed_test?: SubmissionResultResponse;
}

/** dto.ListMySubmissionsResponse — dto/submission.go */
export type ListMySubmissionsResponse = PaginatedList<SubmissionResponse>;

/** dto.ListSubmissionsResponse — dto/submission.go */
export type ListSubmissionsResponse = PaginatedList<SubmissionResponse>;
