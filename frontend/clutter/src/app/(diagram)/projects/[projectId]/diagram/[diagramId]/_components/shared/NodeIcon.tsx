"use client";

import type { NodeType } from "../types";

export function NodeIcon({ type }: { type: NodeType }) {
  const base = "flex items-center justify-center rounded-lg text-[10px] font-bold h-6 w-6";

  switch (type) {
    case "dynamodb":
      return <div className={`${base} bg-indigo-500/15 text-indigo-200 ring-1 ring-indigo-400/30`}>DB</div>;
    case "s3":
      return <div className={`${base} bg-emerald-500/15 text-emerald-200 ring-1 ring-emerald-400/30`}>S3</div>;
    case "lambda":
      return <div className={`${base} bg-amber-500/15 text-amber-200 ring-1 ring-amber-400/30`}>λ</div>;
    case "apigw":
      return <div className={`${base} bg-yellow-500/15 text-yellow-200 ring-1 ring-yellow-400/30`}>API</div>;
    case "enclosure":
      return <div className={`${base} bg-slate-500/15 text-slate-200 ring-1 ring-slate-400/30`}>ENC</div>;
    case "shape":
      return <div className={`${base} bg-slate-500/15 text-slate-200 ring-1 ring-slate-400/30`}>SH</div>;
    case "custom":
      return <div className={`${base} bg-sky-500/15 text-sky-200 ring-1 ring-sky-400/30`}>*</div>;
    default:
      return <div className={`${base} bg-slate-500/15 text-slate-200 ring-1 ring-slate-400/30`} />;
  }
}
