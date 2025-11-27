"use client";

import Navbar from "@/components/common/Navbar";
import { Button } from "@/components/ui/button";
import { setEngine } from "crypto";
import { signIn } from "next-auth/react";
import { useSession } from "next-auth/react";

export default function LoginPage() {
  const { data: session } = useSession();

  const handleGoogleSignIn = async () => {
    // Note: after signIn the page will redirect, so this console.log
    // may not run reliably. Use session in /dashboard instead.
    await signIn("google", { callbackUrl: "/dashboard" });
    console.log(session?.user);
  };

  return (
    <div className="min-h-screen bg-black text-white flex flex-col pt-20 relative overflow-hidden">
      {/* Fixed frosted navbar with no login button */}
      <Navbar showLogin={false} />

      {/* Background Theme (matches home) */}
      <div className="pointer-events-none absolute inset-0 -z-10">
        {/* Soft gradient wash */}
        <div className="absolute inset-0 bg-gradient-to-br from-cyan-500/[0.05] via-transparent to-indigo-500/[0.08] blur-3xl" />
        {/* Glow orbs */}
        <div className="absolute top-1/4 right-[-5%] w-80 h-80 rounded-full bg-cyan-500/30 blur-3xl" />
        <div className="absolute bottom-1/5 left-[-10%] w-96 h-96 rounded-full bg-indigo-500/25 blur-3xl" />
      </div>

      {/* Login Section */}
      <main className="flex flex-1 items-center justify-center px-6 py-16">
        <div className="w-full max-w-md">
          {/* Login Card */}
          <div className="bg-slate-950/70 backdrop-blur-xl rounded-2xl border border-slate-800/70 p-8 shadow-[0_18px_60px_rgba(0,0,0,0.85)]">
            {/* Header */}
            <div className="text-center mb-8">
              <p className="text-xs font-semibold tracking-[0.25em] text-cyan-300 uppercase mb-3">Clutter Access</p>
              <h1 className="text-3xl md:text-4xl font-bold mb-2 tracking-tight">Welcome back</h1>
              <p className="text-sm text-slate-400">Sign in to continue to your projects and diagrams.</p>
            </div>

            {/* Google Login Button */}
            <div className="space-y-4">
              <Button
                onClick={handleGoogleSignIn}
                className="w-full h-12 bg-white hover:bg-slate-100 text-slate-900 font-medium rounded-lg transition-all transform hover:scale-[1.02] flex items-center justify-center gap-3 shadow-[0_10px_40px_rgba(15,23,42,0.45)]"
                variant="outline">
                <svg className="w-5 h-5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 48 48">
                  <path fill="#EA4335" d="M24 9.5c3.54 0 6.71 1.22 9.21 3.6l6.85-6.85C35.9 2.38 30.47 0 24 0 14.62 0 6.51 5.38 2.56 13.22l7.98 6.19C12.6 13.21 17.8 9.5 24 9.5z" />
                  <path fill="#4285F4" d="M46.5 24.5c0-1.54-.15-3.03-.43-4.5H24v9h12.7c-.55 2.96-2.17 5.48-4.61 7.19l7.14 5.57c4.15-3.83 6.57-9.48 6.57-16.26z" />
                  <path fill="#FBBC05" d="M10.54 28.41A14.4 14.4 0 019.5 24c0-1.52.26-2.98.72-4.36l-7.98-6.19A23.94 23.94 0 000 24c0 3.91.94 7.61 2.61 10.91l7.93-6.5z" />
                  <path fill="#34A853" d="M24 48c6.48 0 11.93-2.13 15.9-5.8l-7.14-5.57c-2.06 1.38-4.67 2.18-7.73 2.18-6.2 0-11.4-3.71-13.46-9.02l-7.93 6.5C6.51 42.62 14.62 48 24 48z" />
                </svg>
                Continue with Google
              </Button>
            </div>

            {/* Divider */}
            <div className="relative my-8">
              <div className="absolute inset-0 flex items-center">
                <div className="w-full border-t border-slate-800" />
              </div>
              <div className="relative flex justify-center text-sm">
                <span className="px-4 bg-slate-950/80 text-slate-400">Secure sign in powered by Google</span>
              </div>
            </div>

            {/* Footer Text */}
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
