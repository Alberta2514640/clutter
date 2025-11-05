"use client";
import React, { createContext, useContext, useState, ReactNode, useEffect } from "react";

/** ---- Types ---- */
export type User = {
  id: string;
  name: string;
  role: "owner" | "editor" | "viewer";
};

type AuthContextValue = {
  user: User | null;
  loading: boolean;
};

/** ---- Context ---- */
const AuthContext = createContext<AuthContextValue | undefined>(undefined);

/** ---- Provider ---- */
export function AuthProvider({ children }: { children: ReactNode }) {
  // ✅ Static user for now — no useEffect, so no cascading-render warning
  const [user] = useState<User>({
    id: "me",
    name: "John",
    role: "owner",
  });
  const [loading] = useState(false);

  /**
   * 🚀 When you add real auth (e.g., Google/OAuth), replace the block above with this pattern:
   *
   * const [user, setUser] = useState<User | null>(null);
   * const [loading, setLoading] = useState(true);
   *
   * useEffect(() => {
   *   let isMounted = true;
   *
   *   // Example with a hypothetical auth client:
   *   // 1) Get current session (async)
   *   (async () => {
   *     try {
   *       const session = await authClient.getSession();
   *       if (!isMounted) return;
   *       setUser(session ? { id: session.user.id, name: session.user.name, role: "owner" } : null);
   *     } finally {
   *       if (isMounted) setLoading(false);
   *     }
   *   })();
   *
   *   // 2) Subscribe to auth changes (external callback -> safe to call setState)
   *   const unsub = authClient.onAuthStateChange((session) => {
   *     if (!isMounted) return;
   *     setUser(session ? { id: session.user.id, name: session.user.name, role: "owner" } : null);
   *     setLoading(false);
   *   });
   *
   *   return () => {
   *     isMounted = false;
   *     unsub?.();
   *   };
   * }, []);
   */

  return <AuthContext.Provider value={{ user, loading }}>{children}</AuthContext.Provider>;
}

/** ---- Hook ---- */
export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within an AuthProvider");
  return ctx;
}
