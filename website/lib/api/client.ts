import type { ApiResponse } from "@/types/api";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

type RequestInitWithJson = RequestInit & {
  json?: unknown;
};

export async function apiRequest<T>(path: string, init: RequestInitWithJson = {}): Promise<ApiResponse<T>> {
  const { json, headers, ...rest } = init;

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...rest,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...headers,
    },
    body: json !== undefined ? JSON.stringify(json) : rest.body,
  });

  return response.json() as Promise<ApiResponse<T>>;
}
