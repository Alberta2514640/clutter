import { DemoClip } from "./DemoClip";
import motion from "./motion.module.css";

const steps = ["Drag a VM and a network onto the canvas.", "Connect the VM to the network.", "Save, add a deployment to an AWS connection, and press Deploy.", "Follow the run log as Terraform starts."];

// A short muted clip of the whole path, recorded from the running app.
export function Demo() {
  return (
    <section id="demo" aria-labelledby="demo-title" className="scroll-mt-16 bg-gradient-to-b from-[#020514] to-black">
      <div className="mx-auto grid max-w-6xl items-center gap-10 px-4 py-16 md:px-6 md:py-20 lg:grid-cols-[2fr_3fr]">
        <div className={motion.reveal}>
          <p className="mb-3 text-sm font-semibold uppercase tracking-[0.2em] text-cyan-300">See it work</p>
          <h2 id="demo-title" className="text-3xl font-semibold tracking-tight text-white md:text-4xl">
            Two blocks to a running deploy
          </h2>
          <ol id="demo-steps" className="mt-6 space-y-3 text-slate-300">
            {steps.map((s, i) => (
              <li key={s} className="flex gap-3">
                <span aria-hidden="true" className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-teal-400/50 text-xs font-semibold text-teal-200">
                  {i + 1}
                </span>
                <span className="leading-relaxed">{s}</span>
              </li>
            ))}
          </ol>
          <p className="mt-6 text-sm text-slate-400">Recorded in Clutter, about 18 seconds, with no sound.</p>
        </div>
        <div className={motion.reveal}>
          <DemoClip describedBy="demo-steps" />
        </div>
      </div>
    </section>
  );
}
