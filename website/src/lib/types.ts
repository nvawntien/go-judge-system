/**
 * Types mirroring the Go backend DTOs exposed through the KrakenD gateway.
 *
 * auth service       services/auth/internal/application/dto
 * problem service    services/problem/internal/application/dto
 * submission service services/submission/internal/application/dto
 */

/** pkg/response.APIResponse */
export interface ApiEnvelope<T> {
  status: 'success' | 'error';
  code: number;
  msg?: string;
  data?: T;
}

/* ------------------------------------------------------------------ auth */

export type Role = 'user' | 'contributor' | 'moderator' | 'admin';

export interface LoginRequest {
  identifier: string;
  password: string;
}

export interface RegisterRequest {
  full_name: string;
  username: string;
  email: string;
  password: string;
}

export interface ChangePasswordRequest {
  current_password: string;
  new_password: string;
  confirm_password: string;
}

/* ------------------------------------------------------------------ user */

export interface Me {
  id: string;
  full_name: string;
  username: string;
  email: string;
  role: Role;
  rating: number;
  is_active: boolean;
  avatar_url?: string | null;
  bio?: string | null;
  country?: string | null;
  school?: string | null;
  company?: string | null;
  github_url?: string | null;
  website_url?: string | null;
  linkedin_url?: string | null;
  created_at: string;
  updated_at: string;
}

export interface PublicProfile {
  full_name: string;
  username: string;
  rating: number;
  avatar_url?: string | null;
  bio?: string | null;
  country?: string | null;
  school?: string | null;
  company?: string | null;
  github_url?: string | null;
  website_url?: string | null;
  linkedin_url?: string | null;
  created_at: string;
}

export interface UpdateProfileRequest {
  full_name?: string | null;
  avatar_url?: string | null;
  bio?: string | null;
  country?: string | null;
  school?: string | null;
  company?: string | null;
  github_url?: string | null;
  website_url?: string | null;
  linkedin_url?: string | null;
}

/* --------------------------------------------------------------- problem */

export type Difficulty = 'easy' | 'medium' | 'hard';

export interface Tag {
  id: number;
  name: string;
  slug: string;
  description?: string;
}

export interface ProblemExample {
  input: string;
  output: string;
  explanation?: string;
}

export interface Problem {
  id: number;
  slug: string;
  title: string;
  description: string;
  difficulty: Difficulty;
  tags?: Tag[];
  examples?: ProblemExample[];
  constraints?: string[];
  hints?: string[];
  time_limit: number;
  memory_limit: number;
  author_id?: string;
  is_hidden?: boolean;
  created_at: string;
}

export interface ListProblemsParams {
  page?: number;
  limit?: number;
  difficulty?: Difficulty | '';
  search?: string;
  tag_slug?: string;
}

export interface ListProblemsResponse {
  items: Problem[];
  total: number;
  page: number;
  limit: number;
}

export interface ListTagsResponse {
  items: Tag[];
}

/* ------------------------------------------------------------ submission */

/** services/submission/internal/domain/entity.Status */
export type SubmissionStatus =
  | 'PENDING'
  | 'JUDGING'
  | 'ACCEPTED'
  | 'WRONG_ANSWER'
  | 'TIME_LIMIT_EXCEEDED'
  | 'MEMORY_LIMIT_EXCEEDED'
  | 'RUNTIME_ERROR'
  | 'COMPILATION_ERROR'
  | 'SYSTEM_ERROR';

/** entity.Language values that entity.IsExecutable() accepts. */
export type LanguageCode = 'GO' | 'CPP' | 'PYTHON' | 'JAVA';

export interface CreateSubmissionRequest {
  problem_id: number;
  language: LanguageCode;
  source_code: string;
}

export interface CreateSubmissionResponse {
  id: number;
  problem_id: number;
  problem_title: string;
  language: string;
  status: SubmissionStatus;
  created_at: string;
}

export interface Submission {
  id: number;
  problem_id: number;
  problem_title: string;
  user_id: string;
  username: string;
  language: string;
  source_code: string;
  status: SubmissionStatus;
  created_at: string;
  updated_at: string;
}

export interface SubmissionListItem {
  id: number;
  problem_id: number;
  problem_title: string;
  language: string;
  status: SubmissionStatus;
  created_at: string;
}

export interface Pagination {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

export interface ListSubmissionsResponse {
  items: SubmissionListItem[];
  pagination: Pagination;
}

export interface ListSubmissionsParams {
  page?: number;
  limit?: number;
  status?: SubmissionStatus | '';
  language?: LanguageCode | '';
  problem_id?: number;
}

/* ------------------------------------------------------------------- run */

/**
 * Shape the workspace expects from the (not yet implemented) custom-test run
 * endpoint. Kept here so wiring it up later is a one-file change.
 */
export interface RunTestResult {
  name: string;
  input: string;
  output: string;
  expected?: string;
  passed: boolean;
  time_ms?: number;
}

export interface RunResponse {
  tests: RunTestResult[];
  stdout?: string;
  stderr?: string;
  compile_output?: string;
}
