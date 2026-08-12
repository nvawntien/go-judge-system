import type {
  AdminUser,
  AdminUserListParams,
  AdminUserListResponse,
  AdminListProblemsParams,
  AdminListProblemsResponse,
  AdminListTagsResponse,
  AdminProblemDetail,
  AdminSubmissionDetail,
  AssignUserRoleRequest,
  ApiEnvelope,
  ChangePasswordRequest,
  CreateAdminProblemRequest,
  CreateAdminTagRequest,
  CreateSubmissionRequest,
  CreateSubmissionResponse,
  ListAdminSubmissionsParams,
  ListAdminSubmissionsResponse,
  ListProblemsParams,
  ListProblemsResponse,
  ListSubmissionsParams,
  ListSubmissionsResponse,
  ListTagsResponse,
  LoginRequest,
  Me,
  MyProfileStats,
  Problem,
  PublicProfile,
  RegisterRequest,
  RejudgeAdminSubmissionResponse,
  RunCodeRequest,
  RunResponse,
  Submission,
  SubmissionStreamTicketResponse,
  UpdateAdminProblemRequest,
  UpdateAdminTagRequest,
  UpdateProfileRequest,
  VerifyEmailRequest,
} from './types';

export const API_BASE_URL = (
  process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080'
).replace(/\/$/, '');

export const RUN_ENDPOINT = process.env.NEXT_PUBLIC_RUN_ENDPOINT ?? '/api/v1/submissions/run';

/** Error carrying the backend business code (pkg/response codes.go). */
export class ApiError extends Error {
  readonly httpStatus: number;
  readonly code: number;

  constructor(message: string, httpStatus: number, code: number) {
    super(message);
    this.name = 'ApiError';
    this.httpStatus = httpStatus;
    this.code = code;
  }

  get isUnauthorized() {
    return this.httpStatus === 401;
  }
  get isNotFound() {
    return this.httpStatus === 404;
  }
  /** The endpoint does not exist on the gateway yet. */
  get isUnimplemented() {
    return this.httpStatus === 404 || this.httpStatus === 405 || this.httpStatus === 501;
  }
}

export class NetworkError extends Error {
  constructor(message = 'Network unreachable') {
    super(message);
    this.name = 'NetworkError';
  }
}

type Query = Record<string, string | number | boolean | undefined | null>;

function buildUrl(path: string, query?: Query): string {
  const url = new URL(API_BASE_URL + path);
  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value === undefined || value === null || value === '') continue;
      url.searchParams.set(key, String(value));
    }
  }
  return url.toString();
}

interface RequestOptions {
  method?: string;
  query?: Query;
  body?: unknown;
  /** Send as multipart instead of JSON. */
  formData?: FormData;
  signal?: AbortSignal;
  /** Skip the refresh-token retry (used by auth endpoints themselves). */
  noRetry?: boolean;
}

let refreshInFlight: Promise<boolean> | null = null;

/**
 * The gateway validates a JWT taken from the `access_token` cookie
 * (gateway/settings/*.json → auth/validator.cookie_key). Cookies are HttpOnly,
 * so every request just needs `credentials: 'include'`; on a 401 we try the
 * refresh endpoint once and replay the original request.
 */
async function refreshSession(): Promise<boolean> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      try {
        const res = await fetch(buildUrl('/api/v1/auth/refresh-token'), {
          method: 'POST',
          credentials: 'include',
          headers: { Accept: 'application/json' },
        });
        return res.ok;
      } catch {
        return false;
      } finally {
        // Allow a fresh attempt on the next 401 wave.
        setTimeout(() => {
          refreshInFlight = null;
        }, 0);
      }
    })();
  }
  return refreshInFlight;
}

async function rawRequest(path: string, options: RequestOptions): Promise<Response> {
  const headers: Record<string, string> = { Accept: 'application/json' };
  let body: BodyInit | undefined;

  if (options.formData) {
    body = options.formData;
  } else if (options.body !== undefined) {
    headers['Content-Type'] = 'application/json';
    body = JSON.stringify(options.body);
  }

  try {
    return await fetch(buildUrl(path, options.query), {
      method: options.method ?? 'GET',
      credentials: 'include',
      headers,
      body,
      signal: options.signal,
      cache: 'no-store',
    });
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err;
    throw new NetworkError();
  }
}

export async function apiRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
  let res = await rawRequest(path, options);

  if (res.status === 401 && !options.noRetry) {
    const refreshed = await refreshSession();
    if (refreshed) res = await rawRequest(path, options);
  }

  const text = await res.text();
  let payload: ApiEnvelope<T> | null = null;
  if (text) {
    try {
      payload = JSON.parse(text) as ApiEnvelope<T>;
    } catch {
      payload = null;
    }
  }

  if (!res.ok) {
    throw new ApiError(
      payload?.msg || res.statusText || 'Request failed',
      res.status,
      payload?.code ?? res.status * 100,
    );
  }

  return (payload?.data ?? (payload as unknown)) as T;
}

/* ------------------------------------------------------------------ auth */

export const authApi = {
  login: (body: LoginRequest) =>
    apiRequest<void>('/api/v1/auth/login', { method: 'POST', body, noRetry: true }),

  register: (body: RegisterRequest) =>
    apiRequest<void>('/api/v1/auth/register', { method: 'POST', body, noRetry: true }),

  logout: () => apiRequest<void>('/api/v1/auth/logout', { method: 'POST', noRetry: true }),

  logoutAll: () => apiRequest<void>('/api/v1/auth/logout-all', { method: 'POST' }),

  forgotPassword: (email: string) =>
    apiRequest<void>('/api/v1/auth/password/forgot', {
      method: 'POST',
      body: { email },
      noRetry: true,
    }),

  resetPassword: (body: { token: string; new_password: string; confirm_password: string }) =>
    apiRequest<void>('/api/v1/auth/password/reset', { method: 'POST', body, noRetry: true }),

  changePassword: (body: ChangePasswordRequest) =>
    apiRequest<void>('/api/v1/auth/password/change', { method: 'PUT', body }),

  verifyEmail: (body: VerifyEmailRequest, signal?: AbortSignal) =>
    apiRequest<void>('/api/v1/auth/email/verify', { method: 'POST', body, signal, noRetry: true }),

  resendVerification: (email: string, signal?: AbortSignal) =>
    apiRequest<void>('/api/v1/auth/email/resend-verification', {
      method: 'POST',
      body: { email },
      signal,
      noRetry: true,
    }),
};

/* ------------------------------------------------------------------ user */

export const userApi = {
  me: (signal?: AbortSignal) => apiRequest<Me>('/api/v1/me', { signal, noRetry: false }),

  updateProfile: (body: UpdateProfileRequest) =>
    apiRequest<Me>('/api/v1/me/profile', { method: 'PATCH', body }),

  uploadAvatar: (file: File) => {
    const formData = new FormData();
    formData.append('avatar', file);
    return apiRequest<{ avatarUrl: string }>('/api/v1/me/avatar', { method: 'POST', formData });
  },

  profile: (username: string, signal?: AbortSignal) =>
    apiRequest<PublicProfile>(`/api/v1/users/${encodeURIComponent(username)}/profile`, { signal }),
};

/* --------------------------------------------------------------- problem */

export const problemApi = {
  list: (params: ListProblemsParams = {}, signal?: AbortSignal) =>
    apiRequest<ListProblemsResponse>('/api/v1/problems', {
      query: {
        page: params.page,
        limit: params.limit,
        difficulty: params.difficulty,
        search: params.search,
        tag_slug: params.tag_slug,
      },
      signal,
    }),

  bySlug: (slug: string, signal?: AbortSignal) =>
    apiRequest<Problem>(`/api/v1/problems/${encodeURIComponent(slug)}`, { signal }),

  tags: (signal?: AbortSignal) => apiRequest<ListTagsResponse>('/api/v1/tags', { signal }),
};

/* --------------------------------------------------------------- admin */

export const adminProblemApi = {
  list: (params: AdminListProblemsParams = {}, signal?: AbortSignal) =>
    apiRequest<AdminListProblemsResponse>('/api/v1/admin/problems', {
      query: {
        page: params.page,
        limit: params.limit,
        difficulty: params.difficulty,
        search: params.search,
        tag_slug: params.tag_slug,
      },
      signal,
    }),

  get: (id: number, signal?: AbortSignal) =>
    apiRequest<AdminProblemDetail>(`/api/v1/admin/problems/${id}`, { signal }),

  create: (body: CreateAdminProblemRequest) =>
    apiRequest<Problem>('/api/v1/admin/problems', { method: 'POST', body }),

  update: (id: number, body: UpdateAdminProblemRequest) =>
    apiRequest<Problem>(`/api/v1/admin/problems/${id}`, { method: 'PUT', body }),

  publish: (id: number) =>
    apiRequest<Problem>(`/api/v1/admin/problems/${id}/publish`, { method: 'PATCH' }),

  setHidden: (id: number) =>
    apiRequest<Problem>(`/api/v1/admin/problems/${id}/hidden`, { method: 'PATCH' }),

  delete: (id: number) =>
    apiRequest<void>(`/api/v1/admin/problems/${id}`, { method: 'DELETE' }),

  getTestCase: (problemId: number, signal?: AbortSignal) =>
    apiRequest<AdminProblemDetail['testcase']>(`/api/v1/admin/problems/${problemId}/testcases`, {
      signal,
    }),

  uploadTestCase: (problemId: number, file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return apiRequest<AdminProblemDetail['testcase']>(`/api/v1/admin/problems/${problemId}/testcases`, {
      method: 'POST',
      formData,
    });
  },

  deleteTestCase: (problemId: number) =>
    apiRequest<void>(`/api/v1/admin/problems/${problemId}/testcases`, { method: 'DELETE' }),
};

export const adminTagApi = {
  list: (signal?: AbortSignal) => apiRequest<AdminListTagsResponse>('/api/v1/admin/tags', { signal }),

  create: (body: CreateAdminTagRequest) =>
    apiRequest<AdminListTagsResponse['items'][number]>('/api/v1/admin/tags', { method: 'POST', body }),

  update: (id: number, body: UpdateAdminTagRequest) =>
    apiRequest<AdminListTagsResponse['items'][number]>(`/api/v1/admin/tags/${id}`, {
      method: 'PUT',
      body,
    }),

  delete: (id: number) => apiRequest<void>(`/api/v1/admin/tags/${id}`, { method: 'DELETE' }),
};

export const adminSubmissionApi = {
  list: (params: ListAdminSubmissionsParams = {}, signal?: AbortSignal) =>
    apiRequest<ListAdminSubmissionsResponse>('/api/v1/admin/submissions', {
      query: {
        page: params.page,
        limit: params.limit,
        status: params.status,
        language: params.language,
        problem_id: params.problem_id,
        user_id: params.user_id,
      },
      signal,
    }),

  get: (id: number, signal?: AbortSignal) =>
    apiRequest<AdminSubmissionDetail>(`/api/v1/admin/submissions/${id}`, { signal }),

  rejudge: (id: number) =>
    apiRequest<RejudgeAdminSubmissionResponse>(`/api/v1/admin/submissions/${id}/rejudge`, { method: 'POST' }),
};

export const adminUserApi = {
  list: (params: AdminUserListParams = {}, signal?: AbortSignal) =>
    apiRequest<AdminUserListResponse>('/api/v1/admin/users', {
      query: {
        page: params.page,
        limit: params.limit,
        search: params.search,
        role: params.role,
        status: params.status,
      },
      signal,
    }),

  get: (id: string, signal?: AbortSignal) =>
    apiRequest<AdminUser>(`/api/v1/admin/users/${encodeURIComponent(id)}`, { signal }),

  assignRole: (id: string, body: AssignUserRoleRequest) =>
    apiRequest<void>(`/api/v1/admin/users/${encodeURIComponent(id)}/role`, {
      method: 'PUT',
      body,
    }),

  setSuspension: (id: string, suspended: boolean) =>
    apiRequest<AdminUser>(`/api/v1/admin/users/${encodeURIComponent(id)}/suspension`, {
      method: 'PATCH',
      body: { suspended },
    }),
};

/* ------------------------------------------------------------ submission */

export const submissionApi = {
  create: (body: CreateSubmissionRequest) =>
    apiRequest<CreateSubmissionResponse>('/api/v1/submissions', { method: 'POST', body }),

  get: (id: number, signal?: AbortSignal) =>
    apiRequest<Submission>(`/api/v1/submissions/${id}`, { signal }),

  issueStreamTicket: (id: number, signal?: AbortSignal) =>
    apiRequest<SubmissionStreamTicketResponse>(`/api/v1/submissions/${id}/events/ticket`, {
      method: 'POST',
      signal,
    }),

  listMine: (params: ListSubmissionsParams = {}, signal?: AbortSignal) =>
    apiRequest<ListSubmissionsResponse>('/api/v1/me/submissions', {
      query: {
        page: params.page,
        limit: params.limit,
        status: params.status,
        language: params.language,
        problem_id: params.problem_id,
      },
      signal,
    }),

  getMyProfileStats: (signal?: AbortSignal) =>
    apiRequest<MyProfileStats>('/api/v1/me/profile-stats', { signal }),

  /** Custom-test execution through the submission-service synchronous run API. */
  run: (body: RunCodeRequest) =>
    apiRequest<RunResponse>(RUN_ENDPOINT, { method: 'POST', body }),
};
