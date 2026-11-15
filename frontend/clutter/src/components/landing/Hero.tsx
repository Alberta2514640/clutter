import Link from "next/link";
import motion from "./motion.module.css";
import { Screenshot } from "./Screenshot";
import { SHOTS } from "./shots";
import { FOCUS_RING } from "./site";

// Soft background shapes: plain CSS, no script. They drift slowly, and stay
// still when the visitor asks for reduced motion.
// The rotation sits on the outer box so the drift's own transform does not undo it.
const shapes = [
  { place: "left-[-12%] top-[10%] h-32 w-[36rem] rotate-12", tint: "from-indigo-500/20" },
  { place: "right-[-8%] top-[38%] h-28 w-[30rem] -rotate-12", tint: "from-slate-400/15" },
  { place: "left-[8%] top-[62%] h-20 w-72 -rotate-6", tint: "from-teal-400/15" },
];

// The entrance only moves things (motion.rise), so the first paint is complete.
const delay = (ms: number) => ({ "--delay": `${ms}ms` }) as React.CSSProperties;

export function Hero() {
  return (
    <section aria-labelledby="hero-title" className="relative overflow-hidden bg-[#020514]">
      <div aria-hidden="true" className="pointer-events-none absolute inset-0">
        <div className="absolute inset-0 bg-gradient-to-br from-teal-500/10 via-transparent to-indigo-500/10" />
        {/* Fades into the demo section's colour, so the two read as one flow. */}
        <div className="absolute inset-x-0 bottom-0 h-40 bg-gradient-to-b from-transparent to-[#020514]" />
        {shapes.map((s, i) => (
          <div key={s.place} className={`absolute ${s.place}`}>
            <div className={`h-full w-full rounded-full border border-white/10 bg-gradient-to-r to-transparent ${s.tint} ${motion.drift}`} style={{ animationDelay: `${i * -6}s` }} />
          </div>
        ))}
      </div>

      <div className="relative mx-auto max-w-6xl px-4 pb-12 pt-16 md:px-6 md:pb-16 md:pt-24">
        <div className="mx-auto max-w-3xl text-center">
          <p className={`mb-6 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-3 py-1 text-sm text-slate-300 ${motion.rise}`}>
            <span aria-hidden="true" className="h-2 w-2 rounded-full bg-teal-400" />
            For AWS, OpenStack and private clouds
          </p>
          <h1 id="hero-title" className={`text-4xl font-bold tracking-tight text-white sm:text-6xl md:text-7xl ${motion.rise}`} style={delay(60)}>
            One diagram.
            <span className="block bg-gradient-to-r from-indigo-300 via-white to-teal-300 bg-clip-text pb-2 text-transparent">Every cloud you link.</span>
          </h1>
          <p className={`mx-auto mt-6 max-w-2xl text-lg leading-relaxed text-slate-300 md:text-xl ${motion.rise}`} style={delay(120)}>
            Draw your system once in building blocks. Clutter deploys it to AWS and to any OpenStack cloud, public, private or in Canada, inside accounts you own.
          </p>
          <div className={`mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row ${motion.rise}`} style={delay(180)}>
            <Link
              href="/login"
              className={`inline-flex items-center justify-center rounded-md bg-teal-400 px-6 py-3 font-medium text-black shadow-[0_10px_30px_rgba(45,212,191,0.25)] transition-[background-color,transform] hover:-translate-y-0.5 hover:bg-teal-300 motion-reduce:transition-none motion-reduce:hover:translate-y-0 ${FOCUS_RING}`}>
              Sign in with Google
            </Link>
            <Link
              href="#demo"
              className={`inline-flex items-center justify-center rounded-md border border-slate-600 px-6 py-3 font-medium text-slate-100 transition-[border-color,transform] hover:-translate-y-0.5 hover:border-slate-400 motion-reduce:transition-none motion-reduce:hover:translate-y-0 ${FOCUS_RING}`}>
              Watch the demo
            </Link>
          </div>
          <p className={`mt-4 text-sm text-slate-400 ${motion.rise}`} style={delay(180)}>
            New here? Your Google account creates your Clutter account.
          </p>
        </div>

        <div className={`mx-auto mt-14 max-w-5xl ${motion.rise}`} style={delay(240)}>
          <Screenshot shot={SHOTS.editor} sizes="(min-width: 1072px) 1024px, calc(100vw - 32px)" hero />
        </div>
      </div>
    </section>
  );
}
