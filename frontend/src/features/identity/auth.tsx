import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";

import { tokenStore } from "@/shared/api/client";

import { identityApi, type User } from "./api";

interface AuthValue {
  user: User | null;
  loading: boolean;
  isTranslator: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (username: string, email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthValue | null>(null);

/**
 * Holds the signed-in user.
 *
 * A stored token is verified against /auth/me on mount rather than trusted, so
 * a token revoked on another device does not leave a phantom session here.
 */
export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    if (!tokenStore.access()) {
      setLoading(false);
      return;
    }
    identityApi
      .me()
      .then((u) => !cancelled && setUser(u))
      .catch(() => {
        tokenStore.clear();
        if (!cancelled) setUser(null);
      })
      .finally(() => !cancelled && setLoading(false));

    return () => {
      cancelled = true;
    };
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const res = await identityApi.login({ email, password });
    tokenStore.set(res.token, res.refresh_token);
    setUser(res.user);
  }, []);

  const register = useCallback(async (username: string, email: string, password: string) => {
    const res = await identityApi.register({ username, email, password });
    tokenStore.set(res.token, res.refresh_token);
    setUser(res.user);
  }, []);

  const logout = useCallback(async () => {
    try {
      await identityApi.logout(tokenStore.refresh());
    } catch {
      // Signing out must succeed locally even when the server call fails.
    }
    tokenStore.clear();
    setUser(null);
  }, []);

  const value = useMemo<AuthValue>(
    () => ({
      user,
      loading,
      isTranslator: user?.roles.includes("translator") ?? false,
      login,
      register,
      logout,
    }),
    [user, loading, login, register, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}
