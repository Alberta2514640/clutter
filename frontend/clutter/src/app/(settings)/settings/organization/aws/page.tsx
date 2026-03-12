"use client";
// TODO: swap in the hooks you need
// import { useSomething } from "@/lib/features/.../hooks";

export default function AWSPage() {
  // const isLoading = false;
  // const isSaving = false;

  return (
    <main>
      <section className="bg-slate-900/70 border border-slate-800 rounded-2xl shadow-xl">

        {/* ── Header ─────────────────────────────────────────── */}
        <div className="border-b border-slate-800 px-6 py-4">
          <h2 className="text-lg font-semibold">AWS Account</h2>
          <p className="text-sm text-slate-400 mt-1 max-w-xl">
            Use this to add and link your AWS account to Clutter via IAM roles.
          </p>
        </div>

        {/* ── Body ───────────────────────────────────────────── */}
        <div className="px-6 py-6">
          <div className="flex items-center gap-3">

            {/* Group 1: Account name + Link account */}
            <div className="flex items-center gap-2 flex-1 min-w-0">
              <input
                type="text"
                placeholder="Account name"
                className="flex-1 min-w-0 bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-teal-500 focus:border-transparent transition"
              />
              <button
                type="button"
                className="shrink-0 px-8 py-2 rounded-lg bg-teal-500 hover:bg-teal-400 text-white text-sm font-medium transition-colors"
              >
                Link account
              </button>
            </div>

            {/* Divider */}
            <div className="w-px self-stretch bg-slate-700 mx-1" />

            {/* Group 2: ARN + Save */}
            <div className="flex items-center gap-2 flex-1 min-w-0">
              <input
                type="text"
                placeholder="ARN"
                className="flex-1 min-w-0 bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-teal-500 focus:border-transparent transition"
              />
              <button
                type="button"
                className="shrink-0 px-8 py-2 rounded-lg bg-slate-700 hover:bg-slate-600 text-slate-100 text-sm font-medium transition-colors"
              >
                Save
              </button>
            </div>

            {/* Divider */}
            <div className="w-px self-stretch bg-slate-700 mx-1" />

            {/* Delete */}
            <button
              type="button"
              className="shrink-0 px-12 py-2 rounded-lg bg-red-600 text-sm font-medium text-white hover:bg-red-500 disabled:opacity-60 transition"
            >
              Delete
            </button>

          </div>
        </div>
      </section>
    </main>
  );
}