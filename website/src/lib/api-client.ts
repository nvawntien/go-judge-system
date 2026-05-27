/**
 * Typed API client — one function per endpoint (plan.md §14).
 * No inline fetch in components. All API calls go through this layer.
 *
 * ZERO-GUESS POLICY: Endpoints match exactly from routers:
 *   - Auth router:       services/auth/internal/adapter/inbound/http/router.go
 *   - Problem router:    services/problem/internal/adapter/inbound/http/router.go
 *   - Submission router: services/submission/internal/adapter/inbound/http/router.go
 */

import type {
  APIResponse,
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  ProfileResponse,
  ProblemResponse,
  ListProblemsResponse,
  CreateProblemRequest,
  CreateProblemResponse,
  UpdateProblemRequest,
  CreateSubmissionRequest,
  SubmissionResponse,
  SubmissionDetailResponse,
  ListMySubmissionsResponse,
  ListSubmissionsResponse,
  VerifyEmailRequest,
  ResendVerificationRequest,
  ForgotPasswordRequest,
  ResetPasswordRequest,
  ChangePasswordRequest,
} from "@/types/api";

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//  CONFIG
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//  CORE FETCH WRAPPER
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

class ApiError extends Error {
  constructor(
    public readonly statusCode: number,
    public readonly apiCode: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function request<T>(
  endpoint: string,
  options: RequestInit = {},
): Promise<T> {
  const url = `${API_BASE}${endpoint}`;

  const headers = new Headers(options.headers || {});
  if (!(options.body instanceof FormData) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const res = await fetch(url, {
    ...options,
    credentials: "include", // send cookies (access_token, refresh_token)
    headers,
  });

  const body: APIResponse<T> = await res.json();

  if (!res.ok || body.status === "error") {
    // If 401 → attempt refresh once (plan.md §13)
    if (res.status === 401 && !endpoint.includes("/refresh-token")) {
      const refreshed = await attemptRefresh();
      if (refreshed) {
        return request<T>(endpoint, options);
      }
    }
    throw new ApiError(res.status, body.code, body.msg || "Unknown error");
  }

  return body.data as T;
}

async function attemptRefresh(): Promise<boolean> {
  try {
    const res = await fetch(`${API_BASE}/api/v1/auth/refresh-token`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
    });
    return res.ok;
  } catch {
    return false;
  }
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//  AUTH API — /api/v1/auth/*
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

export const authApi = {
  login(data: LoginRequest): Promise<LoginResponse> {
    return request<LoginResponse>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  register(data: RegisterRequest): Promise<void> {
    return request<void>("/api/v1/auth/register", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  logout(): Promise<void> {
    return request<void>("/api/v1/auth/logout", {
      method: "POST",
    });
  },

  logoutAll(): Promise<void> {
    return request<void>("/api/v1/auth/logout-all", {
      method: "POST",
    });
  },

  refreshToken(): Promise<LoginResponse> {
    return request<LoginResponse>("/api/v1/auth/refresh-token", {
      method: "POST",
    });
  },

  verifyEmail(data: VerifyEmailRequest): Promise<void> {
    return request<void>("/api/v1/auth/email/verify", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  resendVerification(data: ResendVerificationRequest): Promise<void> {
    return request<void>("/api/v1/auth/email/resend-verification", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  forgotPassword(data: ForgotPasswordRequest): Promise<void> {
    return request<void>("/api/v1/auth/password/forgot", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  resetPassword(data: ResetPasswordRequest): Promise<void> {
    return request<void>("/api/v1/auth/password/reset", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  changePassword(data: ChangePasswordRequest): Promise<void> {
    return request<void>("/api/v1/auth/password/change", {
      method: "PUT",
      body: JSON.stringify(data),
    });
  },

  getProfile(username: string): Promise<ProfileResponse> {
    // TODO: unclear from backend — public profile endpoint not in current router,
    // but documented in auth README. Assuming GET /api/v1/auth/profile/:username
    return request<ProfileResponse>(`/api/v1/auth/profile/${username}`);
  },
};

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//  PROBLEM API — /api/v1/problems/*
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

export const problemApi = {
  list(params?: {
    page?: number;
    limit?: number;
    difficulty?: string;
    search?: string;
  }): Promise<ListProblemsResponse> {
    const qs = new URLSearchParams();
    if (params?.page) qs.set("page", String(params.page));
    if (params?.limit) qs.set("limit", String(params.limit));
    if (params?.difficulty) qs.set("difficulty", params.difficulty);
    if (params?.search) qs.set("search", params.search);
    const q = qs.toString();
    return request<ListProblemsResponse>(
      `/api/v1/problems${q ? `?${q}` : ""}`,
    );
  },

  getBySlug(slug: string): Promise<ProblemResponse> {
    return request<ProblemResponse>(`/api/v1/problems/${slug}`);
  },

  listAdmin(params?: {
    page?: number;
    limit?: number;
    difficulty?: string;
    search?: string;
  }): Promise<ListProblemsResponse> {
    const qs = new URLSearchParams();
    if (params?.page) qs.set("page", String(params.page));
    if (params?.limit) qs.set("limit", String(params.limit));
    if (params?.difficulty) qs.set("difficulty", params.difficulty);
    if (params?.search) qs.set("search", params.search);
    const q = qs.toString();
    return request<ListProblemsResponse>(
      `/api/v1/admin/problems${q ? `?${q}` : ""}`,
    );
  },

  getAdmin(id: number): Promise<ProblemResponse> {
    return request<ProblemResponse>(`/api/v1/admin/problems/${id}`);
  },

  create(data: CreateProblemRequest): Promise<CreateProblemResponse> {
    return request<CreateProblemResponse>("/api/v1/admin/problems", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  update(id: number, data: UpdateProblemRequest): Promise<void> {
    return request<void>(`/api/v1/admin/problems/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    });
  },

  delete(id: number): Promise<void> {
    return request<void>(`/api/v1/admin/problems/${id}`, {
      method: "DELETE",
    });
  },

  publish(id: number): Promise<void> {
    return request<void>(`/api/v1/admin/problems/${id}/publish`, {
      method: "PUT",
    });
  },

  hide(id: number): Promise<void> {
    return request<void>(`/api/v1/admin/problems/${id}/hide`, {
      method: "PUT",
    });
  },

  uploadTestcase(id: number, file: File): Promise<void> {
    const formData = new FormData();
    formData.append("file", file);
    return request<void>(`/api/v1/admin/problems/${id}/testcases`, {
      method: "POST",
      body: formData,
    });
  },
};

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//  SUBMISSION API — /api/v1/submissions/*
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

export const submissionApi = {
  create(
    data: CreateSubmissionRequest,
    idempotencyKey: string,
  ): Promise<SubmissionResponse> {
    return request<SubmissionResponse>("/api/v1/submissions", {
      method: "POST",
      body: JSON.stringify(data),
      headers: {
        "X-Idempotency-Key": idempotencyKey,
      },
    });
  },

  listAll(params?: {
    page?: number;
    limit?: number;
    problem_id?: number;
    user_id?: string;
    status?: string;
    language?: string;
  }): Promise<ListSubmissionsResponse> {
    const qs = new URLSearchParams();
    if (params?.page) qs.set("page", String(params.page));
    if (params?.limit) qs.set("limit", String(params.limit));
    if (params?.problem_id) qs.set("problem_id", String(params.problem_id));
    if (params?.user_id) qs.set("user_id", params.user_id);
    if (params?.status) qs.set("status", params.status);
    if (params?.language) qs.set("language", params.language);
    const q = qs.toString();
    return request<ListSubmissionsResponse>(
      `/api/v1/submissions${q ? `?${q}` : ""}`,
    );
  },

  listByProblem(
    problemId: number,
    params?: { page?: number; limit?: number; status?: string; language?: string },
  ): Promise<ListSubmissionsResponse> {
    const qs = new URLSearchParams();
    if (params?.page) qs.set("page", String(params.page));
    if (params?.limit) qs.set("limit", String(params.limit));
    if (params?.status) qs.set("status", params.status);
    if (params?.language) qs.set("language", params.language);
    const q = qs.toString();
    return request<ListSubmissionsResponse>(
      `/api/v1/problems/id/${problemId}/submissions${q ? `?${q}` : ""}`,
    );
  },

  listMy(params?: {
    page?: number;
    limit?: number;
    status?: string;
    language?: string;
  }): Promise<ListMySubmissionsResponse> {
    const qs = new URLSearchParams();
    if (params?.page) qs.set("page", String(params.page));
    if (params?.limit) qs.set("limit", String(params.limit));
    if (params?.status) qs.set("status", params.status);
    if (params?.language) qs.set("language", params.language);
    const q = qs.toString();
    return request<ListMySubmissionsResponse>(
      `/api/v1/my/submissions${q ? `?${q}` : ""}`,
    );
  },

  getMyDetail(id: number): Promise<SubmissionDetailResponse> {
    return request<SubmissionDetailResponse>(`/api/v1/my/submissions/${id}`);
  },
};

export { ApiError };
