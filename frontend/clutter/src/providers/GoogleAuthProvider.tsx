"use client";

import { GoogleOAuthProvider } from "@react-oauth/google";

// The Google OAuth client the website signs in with. Next writes NEXT_PUBLIC_GOOGLE_CLIENT_ID into
// the browser code when it builds. The log-in Lambda accepts only Google tokens minted for its
// GOOGLE_CLIENT_ID, so the two must match: the cloud image takes it as a build argument
// (frontend/Dockerfile.frontend refuses to build without it), and locally backend/devserver/dev.sh
// passes the id A2's log-in Lambda uses (backend/devserver/internal/gateway/env_test.go checks both).
export const GOOGLE_CLIENT_ID = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID ?? "";

export default function GoogleAuthProvider({ children }: { children: React.ReactNode }) {
  return <GoogleOAuthProvider clientId={GOOGLE_CLIENT_ID}>{children}</GoogleOAuthProvider>;
}
