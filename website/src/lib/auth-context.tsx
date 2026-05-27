"use client";

/**
 * Auth Context — plan.md §13.
 * Manages authentication state via React Context + useReducer.
 */

import {
  createContext,
  useContext,
  useReducer,
  useCallback,
  useEffect,
  type ReactNode,
} from "react";
import type {
  AuthState,
  AuthAction,
  AuthUser,
} from "@/types/state";
import type { LoginRequest, RegisterRequest } from "@/types/api";
import { authApi, ApiError } from "@/lib/api-client";

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//  REDUCER — Pure, no side effects (plan.md §27)
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

function authReducer(state: AuthState, action: AuthAction): AuthState {
  switch (action.type) {
    case "LOGIN_START":
      return { status: "AUTHENTICATING" };
    case "LOGIN_SUCCESS":
      return { status: "AUTHENTICATED", user: action.user };
    case "LOGIN_ERROR":
      return { status: "ERROR", error: action.error };
    case "LOGOUT":
      return { status: "IDLE" };
    case "RESTORE":
      return { status: "AUTHENTICATED", user: action.user };
    default: {
      const _exhaustive: never = action;
      return _exhaustive;
    }
  }
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//  CONTEXT
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

interface AuthContextValue {
  state: AuthState;
  login: (data: LoginRequest) => Promise<void>;
  register: (data: RegisterRequest) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//  PROVIDER
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

function parseJwtPayload(token: string): AuthUser | null {
  try {
    const parts = token.split(".");
    if (parts.length !== 3) return null;
    const payload = JSON.parse(atob(parts[1]));
    return {
      userId: payload.user_id || payload.sub || "",
      username: payload.username || "",
      role: payload.role || "user",
    };
  } catch {
    return null;
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(authReducer, { status: "IDLE" });

  // Attempt to restore session from cookie on mount
  useEffect(() => {
    const cookies = document.cookie.split(";").reduce(
      (acc, c) => {
        const [k, v] = c.trim().split("=");
        acc[k] = v;
        return acc;
      },
      {} as Record<string, string>,
    );

    const token = cookies["access_token"];
    if (token) {
      const user = parseJwtPayload(token);
      if (user) {
        dispatch({ type: "RESTORE", user });
      }
    }
  }, []);

  const login = useCallback(async (data: LoginRequest) => {
    dispatch({ type: "LOGIN_START" });
    try {
      const res = await authApi.login(data);
      const user = parseJwtPayload(res.access_token);
      if (!user) throw new Error("Failed to parse token");
      dispatch({ type: "LOGIN_SUCCESS", user });
    } catch (err) {
      const msg =
        err instanceof ApiError ? err.message : "Login failed";
      dispatch({ type: "LOGIN_ERROR", error: msg });
      throw err;
    }
  }, []);

  const register = useCallback(async (data: RegisterRequest) => {
    await authApi.register(data);
  }, []);

  const logout = useCallback(async () => {
    try {
      await authApi.logout();
    } catch {
      // Logout even if API call fails
    }
    dispatch({ type: "LOGOUT" });
  }, []);

  return (
    <AuthContext.Provider value={{ state, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//  HOOK
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}
