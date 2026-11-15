import Link from "next/link";
import motion from "./motion.module.css";
import { FOCUS_RING } from "./site";

export function FinalCta() {
  return (
    <section aria-labelledby="cta-title" className="border-t border-slate-800/60 bg-black">
      <div className="mx-auto max-w-6xl px-4 py-16 md:px-6 md:py-20">
        <div className={`rounded-3xl border border-teal-400/30 bg-gradient-to-br from-teal-400/10 via-slate-950 to-indigo-500/10 px-6 py-12 text-center md:px-12 ${motion.reveal}`}>
          <h2 id="cta-title" className="text-3xl font-semibold tracking-tight text-white md:text-4xl">
            Draw your first diagram
          </h2>
          <p className="mx-auto mt-4 max-w-xl leading-relaxed text-slate-300">Sign in with Google, link a cloud, and see the Terraform for your system before anything is built.</p>
          <Link
            href="/login"
            className={`mt-8 inline-flex items-center justify-center rounded-md bg-teal-400 px-6 py-3 font-medium text-black shadow-[0_10px_30px_rgba(45,212,191,0.25)] transition-[background-color,transform] hover:-translate-y-0.5 hover:bg-teal-300 motion-reduce:transition-none motion-reduce:hover:translate-y-0 ${FOCUS_RING}`}>
            Sign in with Google
          </Link>
        </div>
      </div>
    </section>
  );
}
