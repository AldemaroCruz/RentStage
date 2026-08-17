"use client";

import {
  createContext,
  ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useRouter } from "next/navigation";
import {
  createUserWithEmailAndPassword,
  signInWithEmailAndPassword,
  signOut,
  updateProfile,
} from "firebase/auth";
import { api, ApiError, resetCSRFToken } from "@/lib/api";
import { ensureFirebasePersistence, getFirebaseAuth } from "@/lib/firebase-client";
import type { AuthMe, Permission } from "@/lib/types";

type AuthContextValue = {
  loading: boolean;
  me: AuthMe | null;
  login: (email: string, password: string) => Promise<AuthMe>;
  signup: (displayName: string, email: string, password: string) => Promise<AuthMe>;
  logout: () => Promise<void>;
  refresh: () => Promise<AuthMe | null>;
  selectWorkspace: (tenantID: string) => Promise<AuthMe>;
  can: (permission: Permission) => boolean;
};

const AuthContext = createContext<AuthContextValue | null>(null);

async function exchangeFirebaseSession(): Promise<AuthMe> {
  const auth = getFirebaseAuth();
  const current = auth.currentUser;
  if (!current) throw new Error("Firebase did not return an authenticated user.");
  const idToken = await current.getIdToken(true);
  const me = await api<AuthMe>("/api/v1/auth/session", {
    method: "POST",
    body: JSON.stringify({ id_token: idToken }),
  });
  await signOut(auth);
  return me;
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [me, setMe] = useState<AuthMe | null>(null);

  const refresh = useCallback(async () => {
    try {
      const result = await api<AuthMe>("/api/v1/auth/me");
      setMe(result);
      return result;
    } catch (reason) {
      if (reason instanceof ApiError && reason.status === 401) {
        setMe(null);
        return null;
      }
      throw reason;
    }
  }, []);

  useEffect(() => {
    refresh().catch(() => setMe(null)).finally(() => setLoading(false));
  }, [refresh]);

  useEffect(() => {
    const unauthorized = () => {
      setMe(null);
      router.replace("/login");
    };
    window.addEventListener("rentstage:unauthorized", unauthorized);
    return () => window.removeEventListener("rentstage:unauthorized", unauthorized);
  }, [router]);

  const login = useCallback(async (email: string, password: string) => {
    await ensureFirebasePersistence();
    const auth = getFirebaseAuth();
    await signInWithEmailAndPassword(auth, email.trim(), password);
    const result = await exchangeFirebaseSession();
    setMe(result);
    resetCSRFToken();
    return result;
  }, []);

  const signup = useCallback(async (displayName: string, email: string, password: string) => {
    await ensureFirebasePersistence();
    const auth = getFirebaseAuth();
    const credential = await createUserWithEmailAndPassword(auth, email.trim(), password);
    await updateProfile(credential.user, { displayName: displayName.trim() });
    const result = await exchangeFirebaseSession();
    setMe(result);
    resetCSRFToken();
    return result;
  }, []);

  const logout = useCallback(async () => {
    try {
      await api<void>("/api/v1/auth/session", { method: "DELETE" });
    } finally {
      try {
        await signOut(getFirebaseAuth());
      } catch {
        // The browser uses server sessions; local Firebase state is best-effort.
      }
      setMe(null);
      resetCSRFToken();
      router.replace("/login");
    }
  }, [router]);

  const selectWorkspace = useCallback(async (tenantID: string) => {
    const result = await api<AuthMe>("/api/v1/auth/select-tenant", {
      method: "POST",
      body: JSON.stringify({ tenant_id: tenantID }),
    });
    setMe(result);
    router.refresh();
    return result;
  }, [router]);

  const permissions = useMemo(() => new Set(me?.permissions || []), [me]);
  const can = useCallback((permission: Permission) => permissions.has(permission), [permissions]);

  const value = useMemo<AuthContextValue>(() => ({
    loading,
    me,
    login,
    signup,
    logout,
    refresh,
    selectWorkspace,
    can,
  }), [loading, me, login, signup, logout, refresh, selectWorkspace, can]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) throw new Error("useAuth must be used inside AuthProvider.");
  return context;
}
