"use client";

import { GoogleOAuthProvider } from "@react-oauth/google";

export default function GoogleAuthProvider({ children }: { children: React.ReactNode }) {
  // might wanna move dis to env later
  const clientId = "214630517546-2tttdua57o7gj14up16v7s3unqoah46k.apps.googleusercontent.com";

  if (!clientId) {
    console.error("Missing NEXT_PUBLIC_GOOGLE_CLIENT_ID");
    return <>{children}</>;
  }

  return <GoogleOAuthProvider clientId={clientId}>{children}</GoogleOAuthProvider>;
}
