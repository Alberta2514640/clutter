import { Building2, KeyRound, MapPin, Trash2 } from "lucide-react";
import motion from "./motion.module.css";

const points = [
  {
    icon: Building2,
    title: "Clutter builds only in accounts you link",
    text: "Everything a deployment creates lands in your own cloud account. Clutter never builds your resources in an account of its own.",
  },
  {
    icon: KeyRound,
    title: "Keys are encrypted and never shown again",
    text: "A credential you save is stored encrypted, and Clutter never shows it again, not even to you. For AWS there are no keys: you give Clutter a role, and you can remove it at any time.",
  },
  {
    icon: MapPin,
    title: "Clutter runs in Canada",
    text: "Clutter's servers and database run in Canada (Central). Your diagrams, connections and run logs are stored there. Sign-in goes through Google.",
  },
  {
    icon: Trash2,
    title: "You can remove everything",
    text: "Destroy deletes everything a deployment built. Clutter will not delete a connection while a deployment on it still holds resources.",
  },
];

export function Trust() {
  return (
    <section id="trust" aria-labelledby="trust-title" className="scroll-mt-16 border-t border-slate-800/60 bg-black">
      <div className="mx-auto max-w-6xl px-4 py-16 md:px-6 md:py-24">
        <div className={`mb-10 max-w-2xl ${motion.reveal}`}>
          <p className="mb-3 text-sm font-semibold uppercase tracking-[0.2em] text-cyan-300">Security</p>
          <h2 id="trust-title" className="text-3xl font-semibold tracking-tight text-white md:text-4xl">
            Your accounts stay yours
          </h2>
        </div>
        <ul className={`grid gap-6 sm:grid-cols-2 ${motion.reveal}`}>
          {points.map(({ icon: Icon, title, text }) => (
            <li
              key={title}
              className="flex gap-4 rounded-2xl border border-slate-800 bg-slate-950/70 p-6 transition-[transform,border-color] duration-300 hover:-translate-y-1 hover:border-slate-600 motion-reduce:transition-none motion-reduce:hover:translate-y-0">
              <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-teal-400/40 bg-teal-400/10">
                <Icon className="h-5 w-5 text-teal-300" aria-hidden="true" />
              </span>
              <div>
                <h3 className="mb-2 font-semibold text-white">{title}</h3>
                <p className="text-sm leading-relaxed text-slate-300">{text}</p>
              </div>
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}
