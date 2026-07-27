'use client';

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { ApiError, authApi, userApi } from '@/lib/api';
import type { LoginRequest, Me, RegisterRequest } from '@/lib/types';

interface AuthContextValue {
  user: Me | null;
  /** True until the first /api/v1/me probe settles. */
  loading: boolean;
  login: (body: LoginRequest) => Promise<Me>;
  register: (body: RegisterRequest) => Promise<void>;
  logout: () => Promise<void>;
  refresh: () => Promise<Me | null>;
  setUser: (user: Me | null) => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

/**
 * Session lives in HttpOnly cookies issued by the auth service, so the only way
 * to know who is signed in is to ask the gateway. A 401 here is the normal
 * "signed out" case, not an error worth surfacing.
 */
export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<Me | null>(null);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    try {
      const me = await userApi.me();
      setUser(me);
      return me;
    } catch (err) {
      if (err instanceof ApiError && err.isUnauthorized) {
        setUser(null);
        return null;
      }
      setUser(null);
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const login = useCallback(async (body: LoginRequest) => {
    await authApi.login(body);
    const me = await userApi.me();
    setUser(me);
    return me;
  }, []);

  const register = useCallback(async (body: RegisterRequest) => {
    await authApi.register(body);
  }, []);

  const logout = useCallback(async () => {
    try {
      await authApi.logout();
    } finally {
      setUser(null);
    }
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({ user, loading, login, register, logout, refresh, setUser }),
    [user, loading, login, register, logout, refresh],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used inside <AuthProvider>');
  return ctx;
}
