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

export interface VerifyEmailRequest {
  token: string;
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

export interface PublicUserSearchItem {
  username: string;
  full_name: string;
  avatar_url?: string | null;
  rating: number;
}

export interface SearchUsersParams {
  q: string;
  page?: number;
  limit?: number;
}

export interface SearchUsersResponse {
  items: PublicUserSearchItem[];
  pagination: Pagination;
}

export interface UpdateProfileRequest {
  full_name?: string | null;
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
  expected_output: string;
  explanation?: string | null;
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
  updated_at?: string;
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

export interface ProblemWriteExample {
  input: string;
  expected_output: string;
  explanation?: string;
}

export interface CreateAdminProblemRequest {
  title: string;
  description: string;
  difficulty: Difficulty;
  tag_ids?: number[];
  examples: ProblemWriteExample[];
  constraints: string[];
  hints: string[];
  time_limit: number;
  memory_limit: number;
}

export interface UpdateAdminProblemRequest {
  title?: string;
  slug?: string;
  description?: string;
  difficulty?: Difficulty;
  tag_ids?: number[];
  examples?: ProblemWriteExample[];
  constraints?: string[];
  hints?: string[];
  time_limit?: number;
  memory_limit?: number;
}

export interface AdminTestCaseMetadata {
  has_testcase: boolean;
  id?: number;
  problem_id?: number;
  zip_object_key?: string;
  test_count?: number;
  version?: number;
  created_at?: string;
  updated_at?: string;
}

export interface AdminProblemDetail extends Required<Omit<Problem, 'tags' | 'examples' | 'constraints' | 'hints' | 'updated_at'>> {
  tags: Tag[];
  examples: ProblemExample[];
  constraints: string[];
  hints: string[];
  updated_at: string;
  deleted_at?: string | null;
  testcase: AdminTestCaseMetadata;
}

export interface AdminListProblemsParams extends ListProblemsParams {}

export interface AdminListProblemsResponse {
  items: Problem[];
  total: number;
  page: number;
  limit: number;
}

export interface AdminTag extends Tag {
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface AdminListTagsResponse {
  items: AdminTag[];
}

export interface CreateAdminTagRequest {
  name: string;
  slug?: string;
  description?: string;
  is_active?: boolean;
}

export interface UpdateAdminTagRequest {
  name?: string;
  slug?: string;
  description?: string;
  is_active?: boolean;
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
  | 'OUTPUT_LIMIT_EXCEEDED'
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

export interface SubmissionStreamTicketResponse {
  ticket: string;
  expires_at: string;
}

export interface SubmissionStreamEvent {
  submission_id: number;
  attempt_id: string;
  status: SubmissionStatus;
  updated_at: string;
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
  execution_time_ms: number | null;
  memory_used_kb: number | null;
  passed_testcases: number | null;
  total_testcases: number | null;
  compile_output: string | null;
  error_message: string | null;
  created_at: string;
  updated_at: string;
}

export interface SubmissionListItem {
  id: number;
  problem_id: number;
  problem_title: string;
  language: string;
  status: SubmissionStatus;
  execution_time_ms: number | null;
  memory_used_kb: number | null;
  passed_testcases: number | null;
  total_testcases: number | null;
  created_at: string;
}

export interface AdminSubmissionListItem {
  id: number;
  problem_id: number;
  problem_title: string;
  user_id: string;
  username: string;
  language: string;
  status: SubmissionStatus;
  created_at: string;
}

export interface AdminSubmissionTestResult {
  index: number;
  status: string;
  runtime_ms: number | null;
  memory_kb: number | null;
}

export interface AdminSubmissionDetail {
  id: number;
  problem_id: number;
  problem_title: string;
  user_id: string;
  username: string;
  language: string;
  source_code: string;
  status: SubmissionStatus;
  current_attempt_id: string;
  attempt_trigger: string | null;
  attempt_triggered_by_user_id: string | null;
  attempt_created_at: string | null;
  testcase_version: number | null;
  dataset_checksum: string | null;
  passed_test_count: number;
  executed_test_count: number;
  total_test_count: number | null;
  runtime_ms: number | null;
  memory_kb: number | null;
  compile_message: string | null;
  judge_message: string | null;
  created_at: string;
  updated_at: string;
  test_results: AdminSubmissionTestResult[];
}

export interface RejudgeAdminSubmissionResponse {
  submission_id: number;
  attempt_id: string;
  status: SubmissionStatus;
  attempt_trigger: string;
  attempt_triggered_by_user_id: string;
  enqueued_at: string;
}

export interface ListAdminSubmissionsResponse {
  items: AdminSubmissionListItem[];
  pagination: Pagination;
}

export interface ListAdminSubmissionsParams {
  page?: number;
  limit?: number;
  status?: SubmissionStatus | '';
  language?: LanguageCode | '';
  problem_id?: number;
  user_id?: string;
}

export interface AssignUserRoleRequest {
  role: Role;
}

/* ----------------------------------------------------------- admin users */

export type AdminUserRole = Role;
export type AdminUserStatus = 'active' | 'unverified' | 'suspended';

export interface AdminUser {
  id: string;
  full_name: string;
  username: string;
  email: string;
  role: AdminUserRole;
  rating: number;
  is_active: boolean;
  is_suspended: boolean;
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

export interface AdminUserListParams {
  page?: number;
  limit?: number;
  search?: string;
  role?: AdminUserRole | '';
  status?: AdminUserStatus | '';
}

export interface AdminUserListResponse {
  items: AdminUser[];
  pagination: Pagination;
}

export type SubmissionDetail = Submission;

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

/** Authoritative aggregates returned by Submission for the signed-in user. */
export interface MyProfileStats {
  total_submissions: number;
  attempted_problems: number;
  accepted_submissions: number;
  solved_problems: number;
  acceptance_rate: number;
  verdict_distribution: ProfileStatsVerdict[];
  language_distribution: ProfileStatsLanguage[];
  activity: ProfileStatsActivity[];
}

export interface ProfileStatsVerdict {
  verdict: string;
  count: number;
}

export interface ProfileStatsLanguage {
  language: string;
  count: number;
}

export interface ProfileStatsActivity {
  /** UTC calendar date in YYYY-MM-DD format. */
  date: string;
  count: number;
}

export interface ListSubmissionsParams {
  page?: number;
  limit?: number;
  status?: SubmissionStatus | '';
  language?: LanguageCode | '';
  problem_id?: number;
}

/* ------------------------------------------------------------------- run */

export type RunTestCaseKind = 'sample' | 'custom';

export interface RunTestCaseInput {
  id: string;
  kind: RunTestCaseKind;
  stdin: string;
  expected_output: string | null;
}

export interface RunCodeRequest {
  problem_id: number;
  language: LanguageCode;
  source_code: string;
  testcases: RunTestCaseInput[];
}

export type RunCodeStatus = 'completed' | 'compile_error' | 'system_error';

export interface CodeDiagnostic {
  testcase_id?: string | null;
  kind: 'compile' | 'runtime' | string;
  severity: 'error' | 'warning' | string;
  message: string;
  line: number;
  column: number;
  end_line?: number | null;
  end_column?: number | null;
}

export type RunTestCaseStatus =
  | 'accepted'
  | 'wrong_answer'
  | 'executed'
  | 'runtime_error'
  | 'time_limit_exceeded'
  | 'memory_limit_exceeded'
  | 'output_limit_exceeded'
  | 'system_error';

export interface RunTestCaseResult {
  id: string;
  kind: RunTestCaseKind;
  status: RunTestCaseStatus;
  stdout: string;
  stderr: string;
  expected_output: string | null;
  execution_time_ms: number;
  memory_used_kb: number;
  diagnostics: CodeDiagnostic[];
}

export interface RunCodeResponse {
  status: RunCodeStatus;
  compile_output: string;
  diagnostics: CodeDiagnostic[];
  tests: RunTestCaseResult[];
}

export type RunResponse = RunCodeResponse;
