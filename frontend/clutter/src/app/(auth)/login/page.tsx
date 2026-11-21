"use client";

import ClutterLogo from "@/components/common/ClutterLogo";
import { Button } from "@/components/ui/button";
import { organizationApi } from "@/lib/features/organization/api";
import { homePath } from "@/lib/features/organization/home";
import { useOrganizations } from "@/lib/features/organization/hooks";
import { useLoginWithGoogle, useLogout, useMe } from "@/lib/features/user/hooks";
import { isSessionExpired, SESSION_EXPIRED_NOTICE } from "@/lib/features/user/session";
import type { CredentialResponse } from "@react-oauth/google";
import { AlertTriangle, Info, Loader2 } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import BrandPanel from "./_components/BrandPanel";
import GoogleButton from "./_components/GoogleButton";

function isTokenExpired(token: string) {
  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    return Date.now() >= payload.exp * 1000;
  } catch {
    return true;
  }
}

const SIGN_IN_FAILED = "Clutter could not sign you in. Try again.";

// Sign in and sign up are one page: Google is the only way in (A's log-in
// accepts only Google tokens), and the first sign-in creates the account.
export default function LoginPage() {
  const router = useRouter();

  const login = useLoginWithGoogle();
  const logout = useLogout();
  const meQ = useMe();

  const [signInError, setSignInError] = useState<string | null>(null);
  const [isResolvingLogin, setIsResolvingLogin] = useState(false);
  const [sessionNotice, setSessionNotice] = useState<string | null>(null);
  // Changing the key draws a fresh Google button, which is the retry.
  const [googleButtonKey, setGoogleButtonKey] = useState(0);

  const rawToken = meQ.data?.token ?? null;
  const token = rawToken && !isTokenExpired(rawToken) ? rawToken : null;
  const orgsQ = useOrganizations(token);

  // An expired session elsewhere in the app lands here with ?session=expired.
  useEffect(() => {
    if (new URLSearchParams(window.location.search).get("session") === "expired") {
      setSessionNotice(SESSION_EXPIRED_NOTICE);
    }
  }, []);

  useEffect(() => {
    if (!rawToken || !isTokenExpired(rawToken)) return;
    logout();
    setSessionNotice(SESSION_EXPIRED_NOTICE);
    setSignInError(null);
  }, [rawToken, logout]);

  // Already signed in: go straight to where this person starts.
  useEffect(() => {
    if (token && orgsQ.isSuccess) router.replace(homePath(orgsQ.data));
  }, [token, orgsQ.isSuccess, orgsQ.data, router]);

  const routeAfterLogin = async (authToken: string) => {
    try {
      router.replace(homePath(await organizationApi.list(authToken)));
      return;
    } catch {
      // One quick retry smooths over the transient first-request failure seen after login.
    }
    await new Promise((resolve) => window.setTimeout(resolve, 300));
    router.replace(homePath(await organizationApi.list(authToken)));
  };

  const onCredential = async (response: CredentialResponse) => {
    const googleIdToken = response.credential;
    if (!googleIdToken) {
      setSignInError("Google did not return your sign-in. Try again.");
      return;
    }
    setSignInError(null);
    setSessionNotice(null);
    setIsResolvingLogin(true);
    try {
      const user = await login.mutateAsync(googleIdToken);
      try {
        await routeAfterLogin(user.token);
      } catch (err) {
        if (isSessionExpired(err)) {
          logout();
          setSessionNotice(SESSION_EXPIRED_NOTICE);
        } else {
          setSignInError("You are signed in, but Clutter could not load your teams. Try again.");
        }
      }
    } catch {
      setSignInError(SIGN_IN_FAILED);
    } finally {
      setIsResolvingLogin(false);
    }
  };

  const retryGoogle = () => {
    setSignInError(null);
    setGoogleButtonKey((k) => k + 1);
  };

  // A refused token is handled for the whole app (it signs out and shows the notice).
  const showOrgError = orgsQ.isError && !isSessionExpired(orgsQ.error);
  const busy = (!!token && orgsQ.isLoading) || isResolvingLogin;

  return (
    <div className="grid min-h-dvh lg:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)]">
      <div className="relative isolate flex min-h-dvh flex-col px-4 py-6 sm:px-10 lg:px-16">
        <header>
          <Link href="/" aria-label="Clutter home" className="inline-flex rounded-md outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50">
            <ClutterLogo size={28} withName motion="once" />
          </Link>
        </header>

        <main id="main" className="flex flex-1 items-center py-12">
          <div className="animate-fade-up mx-auto w-full max-w-[400px]">
            <h1 className="text-foreground text-3xl font-semibold tracking-tight">Sign in to Clutter</h1>
            <p className="text-muted-foreground mt-2 text-base">New or returning, it&apos;s one step.</p>

            {sessionNotice && (
              <p role="status" className="border-warning/30 bg-warning/10 text-foreground mt-6 flex gap-2.5 rounded-lg border px-3 py-2.5 text-sm">
                <Info className="text-warning mt-0.5 size-4 shrink-0" aria-hidden="true" />
                {sessionNotice}
              </p>
            )}

            <div className="mt-8">
              <GoogleButton key={googleButtonKey} onCredential={(r) => void onCredential(r)} onError={() => setSignInError("Google sign-in did not finish. Try again.")} />
            </div>
            {/* An email form, when A's log-in takes one, goes here under an "or continue with email" divider. */}

            <p className="text-muted-foreground mt-4 text-center text-sm">New here? This creates your account.</p>

            <div aria-live="polite" className="mt-6 min-h-6">
              {busy && (
                <p role="status" className="text-muted-foreground flex items-center justify-center gap-2 text-sm">
                  <Loader2 className="size-4 animate-spin" aria-hidden="true" />
                  Opening your workspace…
                </p>
              )}
              {!busy && (signInError || showOrgError) && (
                <div role="alert" className="border-danger/25 bg-danger/10 animate-fade-in space-y-3 rounded-lg border p-3 text-sm">
                  <p className="text-foreground flex gap-2.5">
                    <AlertTriangle className="text-danger mt-0.5 size-4 shrink-0" aria-hidden="true" />
                    {signInError ?? "Clutter could not load your teams. Check your connection and try again."}
                  </p>
                  <Button type="button" variant="outline" size="sm" onClick={signInError ? retryGoogle : () => void orgsQ.refetch()}>
                    Try again
                  </Button>
                </div>
              )}
            </div>

            <p className="text-muted-foreground mt-6 text-center text-xs">
              By continuing, you agree to the{" "}
              <Link href="/terms" className="text-foreground underline underline-offset-4 hover:text-link">
                Terms of use
              </Link>{" "}
              and the{" "}
              <Link href="/privacy" className="text-foreground underline underline-offset-4 hover:text-link">
                Privacy policy
              </Link>
              .
            </p>
          </div>
        </main>

        <footer className="text-muted-foreground text-center text-xs lg:text-left">
          Google confirms who you are. Clutter never sees your password.
        </footer>
      </div>

      <BrandPanel />
    </div>
  );
}
