"use client";

import { useState } from "react";

export default function AddMembersCard() {
  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);

  const handleInvite = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    // TODO: wire to backend invite API
    console.log("Invite user:", email);
    setLoading(false);
    setEmail("");
  };

  return (
    <section className="flex flex-col">
      <div className="border-b border-slate-800 px-4 py-3">
        <h2 className="text-sm font-semibold text-slate-100">
          Add users to organization
        </h2>
        <p className="text-xs text-slate-400 mt-1">
          Invite teammates by email. They&apos;ll get access to projects in this
          organization based on their roles.
        </p>
      </div>

      <form onSubmit={handleInvite} className="px-4 py-4 space-y-4 flex-1">
        <div className="space-y-1.5">
          <label className="block text-xs font-medium text-slate-200">
            Email address
          </label>
          <input
            type="email"
            required
            placeholder="teammate@example.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="w-full rounded-lg bg-slate-950/70 border border-slate-800 px-3 py-2 text-sm text-slate-50 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-teal-500 focus:border-teal-500"
          />
          <p className="text-xs text-slate-500">
            Use the same email they use with your IdP (Cognito / SSO).
          </p>
        </div>

        <div className="flex justify-end pt-2">
          <button
            type="submit"
            disabled={loading}
            className="px-3 py-2 rounded-lg bg-teal-500 text-xs font-medium text-black hover:bg-teal-400 disabled:opacity-60 transition"
          >
            {loading ? "Inviting…" : "Invite user"}
          </button>
        </div>
      </form>
    </section>
  );
}
