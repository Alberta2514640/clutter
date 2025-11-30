"use client";

import React from "react";

export default function CanvasLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-gradient-to-br from-black via-slate-900 to-teal-900 text-white">
      {/* Top bar */}
      <header className="sticky top-0 z-50 border-b border-white/10 bg-black/20 backdrop-blur">
        <div className="mx-auto flex w-full max-w-[1600px] items-center justify-between px-6 py-4">
          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={() => window.history.back()}
              className="inline-flex items-center gap-2 rounded-xl border border-white/10 bg-white/5 px-4 py-2 text-sm font-semibold text-slate-100 shadow-sm hover:bg-white/10 active:scale-[0.99]"
            >
              ← Back
            </button>
          </div>

          <button
            type="button"
            // Wire this up later from the page via props/context (e.g., onSave)
            onClick={() => console.log("save")}
            className="inline-flex items-center gap-2 rounded-xl bg-sky-500/20 px-4 py-2 text-sm font-semibold text-sky-100 ring-1 ring-sky-400/30 shadow-sm hover:bg-sky-500/25 active:scale-[0.99]"
          >
            Save
          </button>
        </div>
      </header>

      <main className="mx-auto w-full max-w-[1600px] px-0">
        {/* Keep your page responsible for its own padding */}
        {children}
      </main>
    </div>
  );
}
