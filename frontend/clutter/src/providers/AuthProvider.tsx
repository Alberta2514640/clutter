"use client";
import React, { createContext, useContext, useEffect, useState } from "react";
type User = { id: string; name: string; role?: "owner" | "editor" | "viewer" } | null;
const Ctx = createContext<{ user: User; loading: boolean }>({ user: null, loading: true });
export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User>(null);
  const [loading, setLoading] = useState(true);
  useEffect(() => {
    setUser({ id: "me", name: "Nimna", role: "owner" });
    setLoading(false);
  }, []);
  return <Ctx.Provider value={{ user, loading }}>{children}</Ctx.Provider>;
}
export const useAuth = () => useContext(Ctx);
