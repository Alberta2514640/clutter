"use client";

import { LandingHeader } from "@/components/landing/LandingHeader";
import { Button } from "@/components/ui/button";
import { organizationApi } from "@/lib/features/organization/api";
import { useOrganizations } from "@/lib/features/organization/hooks";
import { useLoginWithGoogle, useLogout, useMe } from "@/lib/features/user/hooks";
import { isSessionExpired, SESSION_EXPIRED_NOTICE } from "@/lib/features/user/session";
import { GoogleLogin } from "@react-oauth/google";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

function isTokenExpired(token: string) {
  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    return Date.now() >= payload.exp * 1000;
  } catch {
    return true;
  }
}

const SIGN_IN_FAILED = "We could not sign you in. Try again.";

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

  const hasOrg = useMemo(() => (orgsQ.data ?? []).length > 0, [orgsQ.data]);

  // An expired session elsewhere in the app lands here with ?session=expired.
  useEffect(() => {
    if (new URLSearchParams(window.location.search).get("session") === "expired") {
      setSessionNotice(SESSION_EXPIRED_NOTICE);
    }
  }, []);

  useEffect(() => {
    if (!rawToken) return;

    if (isTokenExpired(rawToken)) {
      logout();
      setSessionNotice(SESSION_EXPIRED_NOTICE);
      setSignInError(null);
    }
  }, [rawToken, logout]);

  useEffect(() => {
    if (!token) return;
    if (orgsQ.isLoading || orgsQ.isError) return;

    router.replace(hasOrg ? "/dashboard" : "/onboarding/create-org");
  }, [token, orgsQ.isLoading, orgsQ.isError, hasOrg, router]);

  const isChecking = !!token && orgsQ.isLoading;

  const routeAfterLogin = async (authToken: string) => {
    const fetchOrganizations = async () => organizationApi.list(authToken);

    try {
      const orgs = await fetchOrganizations();
      router.replace(orgs.length > 0 ? "/dashboard" : "/onboarding/create-org");
      return;
    } catch {
      // One quick retry smooths over the transient first-request failure seen after login.
    }

    await new Promise((resolve) => window.setTimeout(resolve, 300));

    const orgs = await fetchOrganizations();
    router.replace(orgs.length > 0 ? "/dashboard" : "/onboarding/create-org");
  };

  const retryGoogle = () => {
    setSignInError(null);
    setGoogleButtonKey((k) => k + 1);
  };

  // A refused token is handled for the whole app (it signs out and shows the notice).
  const showOrgError = orgsQ.isError && !isSessionExpired(orgsQ.error);

  return (
    <div className="relative isolate flex min-h-screen flex-col overflow-hidden bg-black text-white">
      <LandingHeader signIn={false} />

      <div className="pointer-events-none absolute inset-0 -z-10">
        <div className="absolute inset-0 bg-gradient-to-br from-cyan-500/[0.05] via-transparent to-indigo-500/[0.08] blur-3xl" />
        <div className="absolute top-1/4 right-[-5%] w-80 h-80 rounded-full bg-cyan-500/30 blur-3xl" />
        <div className="absolute bottom-1/5 left-[-10%] w-96 h-96 rounded-full bg-indigo-500/25 blur-3xl" />
      </div>

      <main id="main" className="flex flex-1 items-center justify-center px-4 py-16 md:px-6">
        <div className="w-full max-w-md">
          <div className="bg-slate-950/70 backdrop-blur-xl rounded-2xl border border-slate-800/70 p-8 shadow-[0_18px_60px_rgba(0,0,0,0.85)]">
            <div className="text-center mb-8">
              <p className="text-xs font-semibold tracking-[0.25em] text-cyan-300 uppercase mb-3">Clutter</p>
              <h1 className="text-2xl md:text-3xl font-bold mb-2 tracking-tight">Sign in or create your account with Google</h1>
              <p className="text-sm text-slate-400">The first time you sign in, Clutter creates your account.</p>
            </div>

            {sessionNotice && (
              <p role="status" className="mb-6 rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-center text-sm text-amber-100">
                {sessionNotice}
              </p>
            )}

            <div className="flex justify-center">
              <GoogleLogin
                key={googleButtonKey}
                onSuccess={async (credentialResponse) => {
                  const googleIdToken = credentialResponse.credential;
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
                        setSignInError("You are signed in, but Clutter could not load your organizations. Try again.");
                      }
                    }
                  } catch {
                    setSignInError(SIGN_IN_FAILED);
                  } finally {
                    setIsResolvingLogin(false);
                  }
                }}
                onError={() => setSignInError("Google sign-in did not finish. Try again.")}
              />
            </div>

            {(isChecking || isResolvingLogin) && (
              <div role="status" className="mt-6 text-center text-sm text-slate-400">
                Checking your organizations…
              </div>
            )}

            {(signInError || showOrgError) && (
              <div role="alert" className="mt-6 space-y-3 text-center text-sm text-red-300">
                <p>{signInError ?? "Clutter could not load your organizations. Check your connection and try again."}</p>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="border-slate-700 bg-slate-900 text-slate-100 hover:bg-slate-800"
                  onClick={signInError ? retryGoogle : () => void orgsQ.refetch()}>
                  Try again
                </Button>
              </div>
            )}

            <p className="mt-8 text-center text-sm text-slate-400">Google confirms who you are. Clutter never sees your password.</p>
          </div>

          <p className="mt-6 text-center text-xs text-slate-400">
            By signing in, you agree to the{" "}
            <Link href="/terms" className="underline underline-offset-4 hover:text-white">
              terms of use
            </Link>
            . The{" "}
            <Link href="/privacy" className="underline underline-offset-4 hover:text-white">
              privacy policy
            </Link>{" "}
            says what Clutter keeps and why.
          </p>
        </div>
      </main>
    </div>
  );
}
