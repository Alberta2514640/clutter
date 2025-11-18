"use client";

import Navbar from "@/components/common/Navbar";
import { Button } from "@/components/ui/button";
import { signIn } from "next-auth/react";

export default function LoginPage() {
  const handleCognitoSignIn = async () => {
    // This hits Cognito Hosted UI (email/password + Google, etc.)
    await signIn("cognito", { callbackUrl: "/dashboard" });
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-black via-slate-900 to-teal-900 text-white">
      {/* Navigation */}
      <Navbar />

      {/* Login Section */}
      <main className="flex flex-col items-center justify-center min-h-screen px-6 py-20">
        <div className="w-full max-w-md">
          {/* Login Card */}
          <div className="bg-slate-900/50 backdrop-blur-xl rounded-2xl border border-slate-800/50 p-8 shadow-2xl">
            {/* Header */}
            <div className="text-center mb-8">
              <h1 className="text-3xl font-bold mb-2">Welcome back</h1>
              <p className="text-gray-400">Sign in to continue to Clutter</p>
            </div>

            {/* Social / SSO Login Buttons */}
            <div className="space-y-4">
              {/* Cognito Hosted UI (email + Google) */}
              <Button onClick={handleCognitoSignIn} className="w-full h-12 bg-white hover:bg-gray-100 text-gray-900 font-medium rounded-lg transition-all transform hover:scale-[1.02]" variant="outline">
                Continue to secure sign in
              </Button>
            </div>

            {/* Divider */}
            <div className="relative my-8">
              <div className="absolute inset-0 flex items-center">
                <div className="w-full border-t border-slate-700"></div>
              </div>
              <div className="relative flex justify-center text-sm">
                <span className="px-4 bg-slate-900/50 text-gray-400">Secure sign in powered by AWS Cognito</span>
              </div>
            </div>

            {/* Footer Text */}
            <p className="text-center text-sm text-gray-400">
              By continuing, you agree to our{" "}
              <a href="/terms" className="text-teal-400 hover:text-teal-300 underline">
                Terms of Service
              </a>{" "}
              and{" "}
              <a href="/privacy" className="text-teal-400 hover:text-teal-300 underline">
                Privacy Policy
              </a>
            </p>
          </div>
        </div>
      </main>

      {/* Background Decoration */}
      <div className="fixed inset-0 pointer-events-none opacity-20 -z-10">
        <div className="absolute top-1/3 right-1/4 w-96 h-96 bg-teal-500 rounded-full filter blur-3xl"></div>
        <div className="absolute bottom-1/3 left-1/4 w-96 h-96 bg-blue-500 rounded-full filter blur-3xl"></div>
      </div>
    </div>
  );
}
