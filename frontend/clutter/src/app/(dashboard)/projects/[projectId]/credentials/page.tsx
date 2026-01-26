"use client";

import { Button } from "@/components/ui/button";
import { useMe } from "@/lib/features/user/hooks";

export default function CredentialsPage() {
  const meQ = useMe();
  const user = meQ.data ?? null;

  const firstName = user?.displayName?.split(" ")[0] || "there";

  return (
    <div className="w-full flex justify-center">
      <div
        className="
          w-full max-w-4xl 
          text-center 
          py-24 
          mt-12
          rounded-2xl
          border border-dashed border-slate-600/50
          bg-slate-900/20
        "
      >
        <div className="text-5xl mb-6">🔑</div>

        <h2 className="text-2xl font-semibold text-white mb-3">
          {firstName}, let&apos;s set up a credential
        </h2>

        <p className="text-gray-400 mb-8">
          Credentials let workflows interact with your apps and services
        </p>

        <Button className="bg-slate-800 border border-slate-700 hover:border-teal-500/50 text-gray-300 px-5 py-4 rounded-lg">
          Add first credential
        </Button>
      </div>
    </div>
  );
}