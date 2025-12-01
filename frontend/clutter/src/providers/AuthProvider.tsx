// "use client";

// import React, { createContext, useContext } from "react";
// import { useSession, SessionProvider } from "next-auth/react";

// type User = {
//   id: string;
//   name: string;
//   email?: string;
//   role?: "owner" | "editor" | "viewer";
// } | null;

// const AuthContext = createContext<{ user: User; loading: boolean }>({
//   user: null,
//   loading: true,
// });

// export function AuthProvider({ children }: { children: React.ReactNode }) {
//   return (
//     <SessionProvider>
//       <AuthInner>{children}</AuthInner>
//     </SessionProvider>
//   );
// }

// // This component extracts session into your custom <AuthContext>
// function AuthInner({ children }: { children: React.ReactNode }) {
//   const { data: session, status } = useSession();

//   const loading = status === "loading";

//   // Convert NextAuth session → your app’s User shape
//   const user: User = session
//     ? {
//         id: session.user?.email ?? "unknown",
//         name: session.user?.name ?? "Anonymous",
//         email: session.user?.email ?? undefined,
//         role: "owner", // TODO: map roles from DB / membership API later
//       }
//     : null;

//   return <AuthContext.Provider value={{ user, loading }}>{children}</AuthContext.Provider>;
// }

// export const useAuth = () => useContext(AuthContext);

// src/providers/AuthProvider.tsx
"use client";

import { SessionProvider } from "next-auth/react";

export function AuthProvider({ children }: { children: React.ReactNode }) {
  return <SessionProvider>{children}</SessionProvider>;
}
