"use client";

import Navbar from "@/components/common/Navbar";
import { useLoginWithGoogle } from "@/lib/features/user/hooks";
import { GoogleLogin } from "@react-oauth/google";
import { useRouter } from "next/navigation";

export default function LoginPage() {
  const router = useRouter();
  const login = useLoginWithGoogle();

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

            <div className="flex justify-center">
              <GoogleLogin
                onSuccess={async (credentialResponse) => {
                  const token = credentialResponse.credential;
                  if (!token) {
                    console.error("No Google ID token returned");
                    return;
                  }

                  //using use mutation from query, better ui states
                  login.mutate(token, {
                    onSuccess: () => router.push("/dashboard"),
                    onError: (err) => console.error(err),
                  });
                }}
                onError={() => console.log("Login Failed")}
              />
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
