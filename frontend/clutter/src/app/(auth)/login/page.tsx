"use client";

import React, { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Navbar from "@/components/common/Navbar";
import { Button } from "@/components/ui/button";

// ---------- GSI Types ----------
type GoogleCredentialResponse = {
  credential: string;
  select_by: string;
  clientId?: string;
};

type GoogleAccountsId = {
  initialize: (options: { client_id: string; callback: (response: GoogleCredentialResponse) => void }) => void;
  prompt: () => void;
};

type GoogleGlobal = {
  accounts: {
    id: GoogleAccountsId;
  };
};

declare global {
  interface Window {
    google?: GoogleGlobal;
  }
}

// ---------- Config ----------
const LOGIN_ENDPOINT = "https://qzq3ncab46.execute-api.us-west-2.amazonaws.com/prod/log-in";

// (later move to env)
const GOOGLE_CLIENT_ID = "214630517546-2tttdua57o7gj14up16v7s3unqoah46k.apps.googleusercontent.com";

export default function LoginPage() {
  const router = useRouter();
  const [gsiReady, setGsiReady] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  // 1) Called by Google when user finishes sign-in
  const handleCredentialResponse = useCallback(
    async (response: GoogleCredentialResponse) => {
      // console.log("✅ GSI callback fired:", response);

      try {
        setIsLoading(true);
        setErrorMsg(null);

        const idToken = response.credential;

        // 🔥 This is the Google ID token
        // console.log("🔥 Google ID Token:", idToken);

        const body = { token: idToken };
        // console.log("📦 Sending to /log-in:", body);

        const res = await fetch(LOGIN_ENDPOINT, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        });

        if (!res.ok) {
          const text = await res.text();
          console.error("Login failed:", text);
          setErrorMsg("Login failed. Please try again.");
          return;
        }

        const data: {
          token: string;
          userData: {
            uuid: string;
            email: string;
            name: string;
            pictureUrl: string;
            accountCreatedOn: string;
          };
          message?: string;
        } = await res.json();

        // console.log("✅ Backend response:", data);

        // Store backend JWT + user data for future calls
        localStorage.setItem("clutter_auth_token", data.token);
        localStorage.setItem("clutter_user", JSON.stringify(data.userData));

        //   Decide where to send them:
        // - first-time user (no org/tenant yet) -> create org page
        // - returning user -> dashboard
        try {
          const profileRes = await fetch("/api/me", {
            method: "GET",
            headers: {
              Authorization: `Bearer ${data.token}`, // or remove if your /api/me reads cookie instead
            },
          });

          if (profileRes.ok) {
            const profile = await profileRes.json();
            const tenantId = profile?.tenantId ?? profile?.tenant?.tenantId ?? null;

            if (!tenantId) {
              router.replace("/onboarding/create-tenant"); // 👈 your CreateTenantPage route
              return;
            }
          }
        } catch {
          // If profile check fails, fall back to home (or onboarding if you prefer)
          router.replace("/");
        }

        // Returning user
        router.replace("/dashboard");
      } catch (err) {
        console.error("Error during login flow:", err);
        setErrorMsg("Something went wrong. Please try again.");
      } finally {
        setIsLoading(false);
      }
    },
    [router]
  );

  // 2) Initialize GSI once the script is available
  useEffect(() => {
    if (!GOOGLE_CLIENT_ID) {
      console.error("Missing GOOGLE_CLIENT_ID");
      return;
    }

    let interval: number | undefined;

    const tryInit = () => {
      if (!window.google) return;
      // console.log("Initializing GSI with client ID:", GOOGLE_CLIENT_ID);
      window.google.accounts.id.initialize({
        client_id: GOOGLE_CLIENT_ID,
        callback: handleCredentialResponse,
      });

      setGsiReady(true);
      // console.log("GSI is ready");
      if (interval !== undefined) window.clearInterval(interval);
    };

    // try immediately
    tryInit();

    // poll until script has loaded
    if (!gsiReady) {
      interval = window.setInterval(tryInit, 300);
    }

    return () => {
      if (interval !== undefined) window.clearInterval(interval);
    };
  }, [handleCredentialResponse, gsiReady]);

  // 3) Trigger Google sign-in prompt
  const handleGoogleSignIn = () => {
    if (!gsiReady || !window.google) {
      setErrorMsg("Google sign-in is not ready yet. Please wait a second and try again.");
      return;
    }

    window.google.accounts.id.prompt();
  };

  // ---------- UI ----------
  return (
    <div className="min-h-screen bg-black text-white flex flex-col pt-20 relative overflow-hidden">
      <Navbar showLogin={false} />

      <div className="pointer-events-none absolute inset-0 -z-10">
        <div className="absolute inset-0 bg-gradient-to-br from-cyan-500/[0.05] via-transparent to-indigo-500/[0.08] blur-3xl" />
        <div className="absolute top-1/4 right-[-5%] w-80 h-80 rounded-full bg-cyan-500/30 blur-3xl" />
        <div className="absolute bottom-1/5 left-[-10%] w-96 h-96 rounded-full bg-indigo-500/25 blur-3xl" />
      </div>

      <main className="flex flex-1 items-center justify-center px-6 py-16">
        <div className="w-full max-w-md">
          <div className="bg-slate-950/70 backdrop-blur-xl rounded-2xl border border-slate-800/70 p-8 shadow-[0_18px_60px_rgba(0,0,0,0.85)]">
            <div className="text-center mb-8">
              <p className="text-xs font-semibold tracking-[0.25em] text-cyan-300 uppercase mb-3">Clutter Access</p>
              <h1 className="text-3xl md:text-4xl font-bold mb-2 tracking-tight">Welcome back</h1>
              <p className="text-sm text-slate-400">Sign in to continue to your projects and diagrams.</p>
            </div>

            <div className="space-y-4">
              <Button
                onClick={handleGoogleSignIn}
                className="w-full h-12 bg-white hover:bg-slate-100 text-slate-900 font-medium rounded-lg transition-all transform hover:scale-[1.02] flex items-center justify-center gap-3 shadow-[0_10px_40px_rgba(15,23,42,0.45)]"
                variant="outline"
                disabled={isLoading}>
                <svg className="w-5 h-5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 48 48">
                  <path fill="#EA4335" d="M24 9.5c3.54 0 6.71 1.22 9.21 3.6l6.85-6.85C35.9 2.38 30.47 0 24 0 14.62 0 6.51 5.38 2.56 13.22l7.98 6.19C12.6 13.21 17.8 9.5 24 9.5z" />
                  <path fill="#4285F4" d="M46.5 24.5c0-1.54-.15-3.03-.43-4.5H24v9h12.7c-.55 2.96-2.17 5.48-4.61 7.19l7.14 5.57c4.15-3.83 6.57-9.48 6.57-16.26z" />
                  <path fill="#FBBC05" d="M10.54 28.41A14.4 14.4 0 019.5 24c0-1.52.26-2.98.72-4.36l-7.98-6.19A23.94 23.94 0 000 24c0 3.91.94 7.61 2.61 10.91l7.93-6.5z" />
                  <path fill="#34A853" d="M24 48c6.48 0 11.93-2.13 15.9-5.8l-7.14-5.57c-2.06 1.38-4.67 2.18-7.73 2.18-6.2 0-11.4-3.71-13.46-9.02l-7.93 6.5C6.51 42.62 14.62 48 24 48z" />
                </svg>
                {isLoading ? "Signing you in..." : "Continue with Google"}
              </Button>

              {errorMsg && <p className="text-xs text-red-400 text-center mt-2">{errorMsg}</p>}
            </div>

            <div className="relative my-8">
              <div className="absolute inset-0 flex items-center">
                <div className="w-full border-t border-slate-800" />
              </div>
              <div className="relative flex justify-center text-sm">
                <span className="px-4 bg-slate-950/80 text-slate-400">Secure sign in powered by Google &amp; Clutter API</span>
              </div>
            </div>

            <p className="text-center text-xs text-slate-400 leading-relaxed">
              By continuing, you agree to our{" "}
              <a href="/terms" className="text-cyan-300 hover:text-cyan-200 underline underline-offset-2">
                Terms of Service
              </a>{" "}
              and{" "}
              <a href="/privacy" className="text-cyan-300 hover:text-cyan-200 underline underline-offset-2">
                Privacy Policy
              </a>
              .
            </p>
          </div>
        </div>
      </main>
    </div>
  );
}
